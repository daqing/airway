package sqlite

import (
	"testing"

	sql "github.com/daqing/airway/lib/sql"
)

func TestInsertBuildsStableSQL(t *testing.T) {
	b := Insert(sql.H{"title": "demo", "completed": true}).Into("todos")

	query, args := b.ToSQL()
	expected := "INSERT INTO todos (completed, title) VALUES (@completed, @title) RETURNING *"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 2 || args["completed"] != true || args["title"] != "demo" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestUpdateWithWhere(t *testing.T) {
	todos := TableOf("todos")
	b := UpdateTable(todos).Set(sql.H{"completed": true}).Where(FieldEq(todos.Field("id"), 1))

	query, args := b.ToSQL()
	if query == "" {
		t.Fatalf("expected non-empty SQL")
	}

	if len(args) == 0 {
		t.Fatalf("expected args, got none")
	}
}
