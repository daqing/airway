package repo

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/daqing/airway/lib/sql"
)

func TestMySQLExecutesILikeFallback(t *testing.T) {
	db := requireMySQLTestDB(t)
	tableName := createTodoTable(t, db)
	todos := sql.TableOf(tableName)
	insertTodoRow(t, db, tableName, "Alpha task", false)

	rows, err := Find[Todo](db, sql.SelectFields(todos.AllFields()).FromTable(todos).
		Where(sql.FieldILike(todos.Field("title"), "%Alpha%")).
		ForUpdate())
	if err != nil {
		t.Fatalf("mysql ilike fallback query: %v", err)
	}

	if len(rows) != 1 || rows[0].Title != "Alpha task" {
		t.Fatalf("unexpected rows: %#v", rows)
	}

	skipLocked, err := Find[Todo](db, sql.SelectFields(todos.AllFields()).FromTable(todos).
		Where(sql.FieldEq(todos.Field("title"), "Alpha task")).
		ForUpdateSkipLocked())
	if err != nil {
		t.Fatalf("mysql skip locked query: %v", err)
	}

	if len(skipLocked) != 1 || skipLocked[0].Title != "Alpha task" {
		t.Fatalf("unexpected skip locked rows: %#v", skipLocked)
	}
}

func TestMySQLInsertRowsWithoutLookupKeyFails(t *testing.T) {
	db := requireMySQLTestDB(t)

	// No AUTO_INCREMENT primary key and no id/conflict values in the rows, so
	// the insert succeeds but the re-select fallback has nothing to look up by.
	tableName := testTableName(t)
	createQuery := fmt.Sprintf(`
CREATE TABLE %s (
	title VARCHAR(255) NOT NULL,
	completed BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`, quoteTestIdentifierForDriver(db.Driver(), tableName))
	if _, err := db.conn.ExecContext(context.Background(), createQuery); err != nil {
		t.Fatalf("create test table: %v", err)
	}

	t.Cleanup(func() {
		dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteTestIdentifierForDriver(db.Driver(), tableName))
		if _, err := db.conn.ExecContext(context.Background(), dropQuery); err != nil {
			t.Fatalf("drop test table: %v", err)
		}
	})

	todos := sql.TableOf(tableName)

	_, err := Insert[Todo](db, sql.InsertRows(
		sql.H{"title": "first", "completed": false},
		sql.H{"title": "second", "completed": true},
	).IntoTable(todos))
	if err == nil {
		t.Fatal("expected multi-row mysql insert fallback to fail without lookup key")
	}

	if !strings.Contains(err.Error(), "retrievable primary or conflict key") {
		t.Fatalf("expected lookup key error, got %v", err)
	}

	if !strings.Contains(err.Error(), "retrievable primary or conflict key") {
		t.Fatalf("unexpected error: %v", err)
	}
}
