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

func TestSelectLockClauses(t *testing.T) {
	todos := TableOf("todos")

	cases := []struct {
		name string
		stmt func() *Builder
		want string
	}{
		{"for update", func() *Builder {
			return SelectFields(todos.AllFields()).FromTable(todos).ForUpdate()
		}, `SELECT "todos".* FROM "todos" FOR UPDATE`},
		{"for share", func() *Builder {
			return SelectFields(todos.AllFields()).FromTable(todos).ForShare()
		}, `SELECT "todos".* FROM "todos" FOR SHARE`},
		{"for update skip locked", func() *Builder {
			return SelectFields(todos.AllFields()).FromTable(todos).ForUpdateSkipLocked()
		}, `SELECT "todos".* FROM "todos" FOR UPDATE SKIP LOCKED`},
		{"raw clause", func() *Builder {
			return SelectFields(todos.AllFields()).FromTable(todos).For("FOR UPDATE NOWAIT")
		}, `SELECT "todos".* FROM "todos" FOR UPDATE NOWAIT`},
	}

	for _, tc := range cases {
		query, _ := tc.stmt().ToSQL()
		if query != tc.want {
			t.Fatalf("%s: expected SQL %q, got %q", tc.name, tc.want, query)
		}
	}
}

func TestWithoutLockingDropsLockClause(t *testing.T) {
	todos := TableOf("todos")
	b := SelectFields(todos.AllFields()).FromTable(todos).ForUpdateSkipLocked()

	query, _ := b.WithoutLocking().ToSQL()
	expected := `SELECT "todos".* FROM "todos"`
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}
}
