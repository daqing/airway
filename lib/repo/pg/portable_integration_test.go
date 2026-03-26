package pg

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/daqing/airway/lib/sql"
	"github.com/jmoiron/sqlx"
)

func TestCountAndExists(t *testing.T) {
	forEachPortableTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTable(t, db)
		todos := sql.TableOf(tableName)
		insertTodoRow(t, db, tableName, "first", true)
		insertTodoRow(t, db, tableName, "second", false)
		insertTodoRow(t, db, tableName, "third", true)

		countBuilder := sql.SelectColumns("count(*)").FromTable(todos).Where(sql.FieldEq(todos.Field("completed"), true))
		count, err := Count(db, countBuilder)
		if err != nil {
			t.Fatalf("count rows: %v", err)
		}

		if count != 2 {
			t.Fatalf("expected 2 completed rows, got %d", count)
		}

		exists, err := Exists(db, sql.SelectColumns("count(*)").FromTable(todos).Where(sql.FieldEq(todos.Field("title"), "second")))
		if err != nil {
			t.Fatalf("exists query: %v", err)
		}

		if !exists {
			t.Fatal("expected row to exist")
		}

		notExists, err := Exists(db, sql.SelectColumns("count(*)").FromTable(todos).Where(sql.FieldEq(todos.Field("title"), "missing")))
		if err != nil {
			t.Fatalf("not exists query: %v", err)
		}

		if notExists {
			t.Fatal("expected row to be absent")
		}
	})
}

func TestFindOneBehaviors(t *testing.T) {
	forEachPortableTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTable(t, db)
		todos := sql.TableOf(tableName)

		missing, err := FindOne[Todo](db, sql.SelectFields(todos.AllFields()).FromTable(todos).Where(sql.FieldEq(todos.Field("title"), "missing")))
		if err != nil {
			t.Fatalf("find missing row: %v", err)
		}

		if missing != nil {
			t.Fatalf("expected nil for missing row, got %#v", missing)
		}

		insertTodoRow(t, db, tableName, "duplicate", false)
		insertTodoRow(t, db, tableName, "duplicate", true)

		_, err = FindOne[Todo](db, sql.SelectFields(todos.AllFields()).FromTable(todos).Where(sql.FieldEq(todos.Field("title"), "duplicate")))
		if !errors.Is(err, ErrorCountNotMatch) {
			t.Fatalf("expected ErrorCountNotMatch, got %v", err)
		}
	})
}

func TestTransactionCommitAndRollback(t *testing.T) {
	forEachPortableTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTable(t, db)
		insertSQL := db.conn.Rebind(fmt.Sprintf("INSERT INTO %s (title, completed) VALUES (?, ?)", quoteTestIdentifierForDriver(db.Driver(), tableName)))

		if err := Tx(db, func(tx *sqlx.Tx) error {
			_, err := tx.ExecContext(context.Background(), insertSQL, "committed", false)
			return err
		}); err != nil {
			t.Fatalf("commit transaction: %v", err)
		}

		rollbackErr := errors.New("force rollback")
		err := Tx(db, func(tx *sqlx.Tx) error {
			if _, execErr := tx.ExecContext(context.Background(), insertSQL, "rolled-back", true); execErr != nil {
				return execErr
			}

			return rollbackErr
		})
		if !errors.Is(err, rollbackErr) {
			t.Fatalf("expected rollback error, got %v", err)
		}

		if count := countRows(t, db, tableName); count != 1 {
			t.Fatalf("expected 1 row after rollback, got %d", count)
		}
	})
}

func TestInsertOnConflictDoNothing(t *testing.T) {
	forEachPortableTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTableWithUniqueTitle(t, db)
		todos := sql.TableOf(tableName)

		original, err := Insert[Todo](db, sql.Insert(sql.H{"title": "same-title", "completed": false}).IntoTable(todos))
		if err != nil {
			t.Fatalf("insert original row: %v", err)
		}

		_, err = Insert[Todo](db, sql.Insert(sql.H{"title": "same-title", "completed": true}).IntoTable(todos).OnConflictDoNothing("title"))
		if err != nil {
			t.Fatalf("insert with do nothing: %v", err)
		}

		if count := countRows(t, db, tableName); count != 1 {
			t.Fatalf("expected 1 row after do nothing, got %d", count)
		}

		row, err := FindOne[Todo](db, sql.SelectFields(todos.AllFields()).FromTable(todos).Where(sql.FieldEq(todos.Field("title"), "same-title")))
		if err != nil {
			t.Fatalf("read existing row: %v", err)
		}

		if row == nil || row.ID != original.ID || row.Completed {
			t.Fatalf("unexpected row after do nothing: %#v", row)
		}
	})
}

func TestInsertOnConflictDoUpdate(t *testing.T) {
	forEachPortableTestDB(t, func(t *testing.T, db *DB) {
		tableName := createTodoTableWithUniqueTitle(t, db)
		todos := sql.TableOf(tableName)

		_, err := Insert[Todo](db, sql.Insert(sql.H{"title": "same-title", "completed": false}).IntoTable(todos))
		if err != nil {
			t.Fatalf("insert original row: %v", err)
		}

		updated, err := Insert[Todo](db, sql.Insert(sql.H{"title": "same-title", "completed": true}).IntoTable(todos).OnConflictDoUpdate([]string{"title"}, sql.H{"completed": true}))
		if err != nil {
			t.Fatalf("upsert row: %v", err)
		}

		if updated == nil || updated.Title != "same-title" || !updated.Completed {
			t.Fatalf("unexpected upserted row: %#v", updated)
		}

		if count := countRows(t, db, tableName); count != 1 {
			t.Fatalf("expected 1 row after upsert, got %d", count)
		}
	})
}

func TestSQLiteExecutesILikeAndForUpdateFallback(t *testing.T) {
	db := requireSQLiteTestDB(t)
	tableName := createTodoTable(t, db)
	todos := sql.TableOf(tableName)
	insertTodoRow(t, db, tableName, "Alpha task", false)
	insertTodoRow(t, db, tableName, "Beta task", true)

	rows, err := Find[Todo](db, sql.SelectFields(todos.AllFields()).FromTable(todos).
		Where(sql.FieldILike(todos.Field("title"), "%Alpha%")).
		OrderBy(todos.Field("id").Asc()).
		ForUpdate())
	if err != nil {
		t.Fatalf("sqlite ilike/lock fallback query: %v", err)
	}

	if len(rows) != 1 || rows[0].Title != "Alpha task" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}
