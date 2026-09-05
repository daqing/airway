package mysql

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

func TestSelectWithWhere(t *testing.T) {
	todos := TableOf("todos")
	b := SelectFields(todos.AllFields()).FromTable(todos).Where(AllOf(
		FieldEq(todos.Field("completed"), false),
	))

	query, args := b.ToSQL()
	expected := `SELECT "todos".* FROM "todos" WHERE "todos"."completed" = @right`
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 1 || args["right"] != false {
		t.Fatalf("unexpected args: %#v", args)
	}
}
