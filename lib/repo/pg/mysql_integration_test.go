package pg

import (
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
}

func TestMySQLInsertRowsWithoutLookupKeyFails(t *testing.T) {
	db := requireMySQLTestDB(t)
	tableName := createTodoTable(t, db)
	todos := sql.TableOf(tableName)

	_, err := Insert[Todo](db, sql.InsertRows(
		sql.H{"title": "first", "completed": false},
		sql.H{"title": "second", "completed": true},
	).IntoTable(todos))
	if err == nil {
		t.Fatal("expected multi-row mysql insert fallback to fail without lookup key")
	}

	if !strings.Contains(err.Error(), "retrievable primary or conflict key") {
		t.Fatalf("unexpected error: %v", err)
	}
}
