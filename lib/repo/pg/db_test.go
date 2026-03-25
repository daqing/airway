package pg

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/daqing/airway/lib/sql"
)

const testDSNEnv = "AIRWAY_PG_TEST_DSN"

type Todo struct {
	ID        int64     `db:"id"`
	Title     string    `db:"title"`
	Completed bool      `db:"completed"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func quoteTestIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func testTableName(t *testing.T) string {
	t.Helper()

	name := strings.ToLower(t.Name())
	name = strings.NewReplacer("/", "_", " ", "_", "-", "_").Replace(name)
	return fmt.Sprintf("airway_%s_%d", name, time.Now().UnixNano())
}

func createTodoTable(t *testing.T, db *DB) string {
	t.Helper()

	tableName := testTableName(t)
	query := fmt.Sprintf(`
CREATE TABLE %s (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`, quoteTestIdentifier(tableName))

	if _, err := db.pool.Exec(context.Background(), query); err != nil {
		t.Fatalf("create test table: %v", err)
	}

	t.Cleanup(func() {
		dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteTestIdentifier(tableName))
		if _, err := db.pool.Exec(context.Background(), dropQuery); err != nil {
			t.Fatalf("drop test table: %v", err)
		}
	})

	return tableName
}

func insertTodoRow(t *testing.T, db *DB, tableName string, title string, completed bool) int64 {
	t.Helper()

	query := fmt.Sprintf(
		"INSERT INTO %s (title, completed) VALUES ($1, $2) RETURNING id",
		quoteTestIdentifier(tableName),
	)

	var id int64
	if err := db.pool.QueryRow(context.Background(), query, title, completed).Scan(&id); err != nil {
		t.Fatalf("insert seed row: %v", err)
	}

	return id
}

func countRows(t *testing.T, db *DB, tableName string) int {
	t.Helper()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteTestIdentifier(tableName))
	var count int
	if err := db.pool.QueryRow(context.Background(), query).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}

	return count
}

func requireTestDB(t *testing.T) *DB {
	t.Helper()

	dsn := os.Getenv(testDSNEnv)
	if dsn == "" {
		t.Skipf("set %s to run PostgreSQL integration tests", testDSNEnv)
	}

	db, err := NewDB(dsn)
	if err != nil {
		t.Skipf("skip PostgreSQL integration tests: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.pool.Ping(ctx); err != nil {
		db.pool.Close()
		t.Skipf("skip PostgreSQL integration tests: %v", err)
	}

	t.Cleanup(func() {
		db.pool.Close()
	})

	return db
}

func TestInsert(t *testing.T) {
	db := requireTestDB(t)
	tableName := createTodoTable(t, db)

	todos := sql.TableOf(tableName)

	var todo *Todo
	b := sql.Insert(sql.H{"title": "test233", "completed": true}).IntoTable(todos)
	todo, err := Insert[Todo](db, b)
	if err != nil {
		t.Fatal(err)
	}

	if todo == nil {
		t.Fatal("todo is nil")
	}

	if todo.Title != "test233" || !todo.Completed || todo.ID == 0 {
		t.Fatalf("unexpected inserted todo: %#v", todo)
	}
}

func TestSelect(t *testing.T) {
	db := requireTestDB(t)
	tableName := createTodoTable(t, db)
	insertTodoRow(t, db, tableName, "select me", false)

	todosTable := sql.TableOf(tableName)

	var todos []*Todo
	b := sql.SelectFields(todosTable.AllFields()).FromTable(todosTable)
	todos, err := Find[Todo](db, b)
	if err != nil {
		t.Fatal(err)
	}

	if len(todos) == 0 {
		t.Fatal("todos is empty")
	}

	if len(todos) != 1 || todos[0].Title != "select me" || todos[0].Completed {
		t.Fatalf("unexpected todos: %#v", todos)
	}
}

func TestUpdate(t *testing.T) {
	db := requireTestDB(t)
	tableName := createTodoTable(t, db)
	insertTodoRow(t, db, tableName, "before update", true)

	todos := sql.TableOf(tableName)
	b := sql.UpdateTable(todos).
		Set(sql.H{"title": "test updated", "completed": false}).
		Where(sql.FieldEq(todos.Field("title"), "before update"))
	err := Update(db, b)
	if err != nil {
		t.Fatalf("failed to update todo")
	}

	updated, err := Find[Todo](db, sql.SelectFields(todos.AllFields()).FromTable(todos))
	if err != nil {
		t.Fatalf("find updated todo: %v", err)
	}

	if len(updated) != 1 || updated[0].Title != "test updated" || updated[0].Completed {
		t.Fatalf("unexpected updated todo: %#v", updated)
	}
}

func TestDelete(t *testing.T) {
	db := requireTestDB(t)
	tableName := createTodoTable(t, db)
	firstID := insertTodoRow(t, db, tableName, "first", false)
	insertTodoRow(t, db, tableName, "second", true)

	todosTable := sql.TableOf(tableName)
	b := sql.DeleteFrom(todosTable).OrderBy(todosTable.Field("id").Asc()).Limit(1)
	err := Delete(db, b)
	if err != nil {
		t.Fatal(err)
	}

	remaining, err := Find[Todo](db, sql.SelectFields(todosTable.AllFields()).FromTable(todosTable).OrderBy(todosTable.Field("id").Asc()))
	if err != nil {
		t.Fatalf("find remaining todos: %v", err)
	}

	if count := countRows(t, db, tableName); count != 1 {
		t.Fatalf("expected 1 remaining row, got %d", count)
	}

	if len(remaining) != 1 || remaining[0].ID == firstID || remaining[0].Title != "second" {
		t.Fatalf("unexpected remaining todos: %#v", remaining)
	}
}
