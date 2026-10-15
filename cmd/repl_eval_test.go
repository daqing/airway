package cmd

import (
	"strings"
	"testing"
)

func TestRepoREPLEvaluatesSQLCompositeLiterals(t *testing.T) {
	session := newTestREPL(t)

	cases := []struct {
		name  string
		expr  string
		query string
	}{
		{
			name:  "positional condition",
			expr:  `repo.Preview(pg.Select("*").Where(sql.Condition{"id", ">", 0}))`,
			query: "WHERE id > ?",
		},
		{
			name:  "keyed condition pointer",
			expr:  `repo.Preview(pg.Select("*").Where(&sql.Condition{Key: "name", Op: "LIKE", Val: "%a%"}))`,
			query: "WHERE name LIKE ?",
		},
		{
			name:  "compare condition with identifier",
			expr:  `repo.Preview(pg.Select("*").Where(sql.CompareCond{Left: sql.Field("id"), Op: "=", Right: 3}))`,
			query: `WHERE "id" = ?`,
		},
		{
			name:  "bool condition over interface slice",
			expr:  `repo.Preview(pg.Select("*").Where(sql.BoolCond{Op: sql.And, Conds: []sql.CondBuilder{sql.Eq("id", 1), sql.Eq("id", 2)}}))`,
			query: "WHERE (id = ? AND id = ?)",
		},
		{
			name:  "and condition with elided pointer elements",
			expr:  `repo.Preview(pg.Select("*").Where(sql.AndCond{Conds: []*sql.Condition{{Key: "id", Op: ">", Val: 0}, {Key: "enabled", Op: "=", Val: true}}}))`,
			query: "WHERE id > ? AND enabled = ?",
		},
		{
			name:  "generic in condition",
			expr:  `repo.Preview(pg.Select("*").Where(sql.InCond[int]{Column: "id", Values: []int{1, 2}}))`,
			query: "WHERE id IN (?, ?)",
		},
		{
			name:  "null condition",
			expr:  `repo.Preview(pg.Select("*").Where(sql.NullCond{Column: "name", Negated: true}))`,
			query: "WHERE name IS NOT NULL",
		},
		{
			name:  "between condition",
			expr:  `repo.Preview(pg.Select("*").Where(sql.BetweenCond{Column: "id", Lower: 1, Upper: 9}))`,
			query: "WHERE id BETWEEN ? AND ?",
		},
		{
			name:  "condition group with constants",
			expr:  `repo.Preview(pg.Select("*").Where(sql.ConditionGroup{Left: sql.Eq("id", 1), Op: sql.And, Right: sql.Eq("enabled", true)}))`,
			query: "WHERE (id = ? AND enabled = ?)",
		},
		{
			name:  "map condition from H literal",
			expr:  `repo.Preview(pg.Select("*").Where(sql.MapCond{Cond: pg.H{"enabled": true}}))`,
			query: "WHERE enabled = ?",
		},
		{
			name:  "not condition",
			expr:  `repo.Preview(pg.Select("*").Where(sql.NotCond{Cond: sql.Eq("id", 1)}))`,
			query: "WHERE NOT (id = ?)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query, _ := previewSQL(t, session, tc.expr)
			if !strings.Contains(query, tc.query) {
				t.Fatalf("query %q does not contain %q", query, tc.query)
			}
		})
	}

	query, args := previewSQL(t, session, `repo.Preview(pg.Select("*").Where(sql.Condition{"id", ">", 7}))`)
	if !strings.Contains(query, "WHERE id > ?") {
		t.Fatalf("unexpected positional condition query: %q", query)
	}

	if len(args) != 1 || args[0] != float64(7) {
		t.Fatalf("unexpected positional condition args: %#v", args)
	}
}

func TestRepoREPLEvaluatesSQLTypeConversionsAndQualifiedLiterals(t *testing.T) {
	session := newTestREPL(t)

	if got := executeAndDecode(t, session, `sql.ConditionGroupOp("AND")`); got != "AND" {
		t.Fatalf("unexpected converted value: %#v", got)
	}

	got := executeAndDecode(t, session, `sql.NamedArgs{"id": 1}`)
	named, ok := got.(map[string]any)
	if !ok || named["id"] != float64(1) {
		t.Fatalf("unexpected named args literal: %#v", got)
	}

	if got := executeAndDecode(t, session, `sql.IdType(42)`); got != float64(42) {
		t.Fatalf("unexpected converted id: %#v", got)
	}
}

func TestRepoREPLFindAcceptsCompositeCondition(t *testing.T) {
	session := newTestREPL(t)

	executeAndDecode(t, session, `repo.Insert("users", pg.H{"name": "carl", "email": "carl@example.com", "enabled": true})`)

	result := executeAndDecode(t, session, `repo.Find("users", pg.Select("*").Where(sql.Condition{"enabled", "=", true}))`)
	rows, ok := result.([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("expected one row from composite condition, got %#v", result)
	}

	row, ok := rows[0].(map[string]any)
	if !ok || row["email"] != "carl@example.com" {
		t.Fatalf("unexpected row: %#v", rows[0])
	}

	// The bare table+condition form must accept a value-typed condition too.
	bare := executeAndDecode(t, session, `repo.Find("users", sql.Condition{"enabled", "=", true})`)
	bareRows, ok := bare.([]any)
	if !ok || len(bareRows) != 1 {
		t.Fatalf("expected one row from bare composite condition, got %#v", bare)
	}
}

func TestRepoREPLStillSupportsHMapLiterals(t *testing.T) {
	session := newTestREPL(t)

	got := executeAndDecode(t, session, `pg.H{"name": "dana", "enabled": true}`)
	values, ok := got.(map[string]any)
	if !ok || values["name"] != "dana" || values["enabled"] != true {
		t.Fatalf("unexpected H literal: %#v", got)
	}
}

func previewSQL(t *testing.T, session *repoREPL, expr string) (string, []any) {
	t.Helper()

	value := executeAndDecode(t, session, expr)
	preview, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected preview map for %q, got %#v", expr, value)
	}

	query, _ := preview["query"].(string)
	args, _ := preview["args"].([]any)
	return query, args
}
