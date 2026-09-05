package pg

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

func TestNestedConditionsGetUniqueNamedArgs(t *testing.T) {
	issues := TableOf("issues")
	b := SelectFields(issues.AllFields()).FromTable(issues).Where(AllOf(
		FieldEq(issues.Field("status"), "open"),
		AnyOf(
			FieldEq(issues.Field("status"), "closed"),
			FieldEq(issues.Field("kind"), "feature"),
		),
	))

	query, args := b.ToSQL()
	expected := `SELECT "issues".* FROM "issues" WHERE ("issues"."status" = @right AND ("issues"."status" = @right_1 OR "issues"."kind" = @right_2))`
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 3 || args["right"] != "open" || args["right_1"] != "closed" || args["right_2"] != "feature" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
