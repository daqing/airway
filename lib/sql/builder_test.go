package sql

import "testing"

func TestInsertStableColumnOrder(t *testing.T) {
	b := Insert(H{"title": "demo", "completed": true}).Into("todos")

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
		Not(IsNull(issues.Field("deleted_at").String())),
	))

	query, args := b.ToSQL()
	expected := "SELECT \"issues\".* FROM \"issues\" WHERE (\"issues\".\"status\" = @right AND (\"issues\".\"status\" = @right_1 OR \"issues\".\"kind\" = @right_2) AND NOT (\"issues\".\"deleted_at\" IS NULL))"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %#v", args)
	}

	if args["right"] != "open" || args["right_1"] != "closed" || args["right_2"] != "feature" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestSelectSupportsWithJoinHavingAndLock(t *testing.T) {
	sessions := TableOf("sessions")
	users := TableAlias("users", "u")
	posts := TableAlias("posts", "p")
	activeUsersRef := TableAlias("active_users", "au")

	activeUsers := SelectFields(Field("user_id")).FromTable(sessions).Where(FieldEq(Field("revoked"), false))

	b := SelectFields(users.Field("id")).
		Columns(`COUNT("p"."id") AS post_count`).
		With("active_users", activeUsers).
		FromTable(users).
		JoinTable(activeUsersRef, FieldEq(activeUsersRef.Field("user_id"), users.Field("id"))).
		LeftJoinTable(posts, FieldEq(posts.Field("user_id"), users.Field("id"))).
		Where(FieldEq(users.Field("enabled"), true)).
		GroupByFields(users.Field("id")).
		Having(Compare(Func("COUNT", posts.Field("id")), ">", 0)).
		OrderBy(users.Field("id").Desc()).
		Limit(20).
		Offset(40).
		ForUpdate()

	query, args := b.ToSQL()
	expected := "WITH active_users AS (SELECT \"user_id\" FROM \"sessions\" WHERE \"revoked\" = @right) SELECT \"u\".\"id\", COUNT(\"p\".\"id\") AS post_count FROM \"users\" AS \"u\" JOIN \"active_users\" AS \"au\" ON \"au\".\"user_id\" = \"u\".\"id\" LEFT JOIN \"posts\" AS \"p\" ON \"p\".\"user_id\" = \"u\".\"id\" WHERE \"u\".\"enabled\" = @right_1 GROUP BY \"u\".\"id\" HAVING COUNT(\"p\".\"id\") > @right_2 ORDER BY \"u\".\"id\" DESC LIMIT 20 OFFSET 40 FOR UPDATE"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 3 || args["right"] != false || args["right_1"] != true || args["right_2"] != 0 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestInsertRowsOnConflictDoUpdate(t *testing.T) {
	b := InsertRows(
		H{"id": 1, "name": "alpha"},
		H{"id": 2, "name": "beta"},
	).Into("users").
		OnConflictDoUpdate([]string{"id"}, H{
			"name":       Excluded("name"),
			"updated_at": Raw("NOW()"),
		}).
		Returning("id", "name")

	query, args := b.ToSQL()
	expected := "INSERT INTO users (id, name) VALUES (@id, @name), (@id_1, @name_1) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, updated_at = NOW() RETURNING id, name"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 4 || args["id"] != 1 || args["name"] != "alpha" || args["id_1"] != 2 || args["name_1"] != "beta" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestSelectSupportsUnionAndWindow(t *testing.T) {
	users := TableOf("users")
	invitations := TableOf("invitations")
	active := SelectFields(Field("id")).FromTable(users).Where(FieldEq(Field("enabled"), true))
	invited := SelectFields(Field("user_id").As("id")).FromTable(invitations).Where(FieldEq(Field("accepted"), true))

	b := active.
		UnionAll(invited).
		Window("w AS (PARTITION BY id)").
		OrderBy(Field("id").Desc()).
		Limit(10)

	query, args := b.ToSQL()
	expected := "SELECT \"id\" FROM \"users\" WHERE \"enabled\" = @right WINDOW w AS (PARTITION BY id) UNION ALL (SELECT \"user_id\" AS \"id\" FROM \"invitations\" WHERE \"accepted\" = @right_1) ORDER BY \"id\" DESC LIMIT 10"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 2 || args["right"] != true || args["right_1"] != true {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestJoinLateralAndExpressionHelpers(t *testing.T) {
	users := TableAlias("users", "u")
	posts := TableAlias("posts", "p")
	lp := TableOf("lp")
	latestPost := SelectFields(posts.Field("user_id"), posts.Field("title")).
		Columns(`ROW_NUMBER() OVER (ORDER BY "p"."created_at" DESC) AS rn`).
		FromTable(posts).
		Where(FieldEq(posts.Field("user_id"), users.Field("id"))).
		Limit(1)

	b := SelectFields(
		users.Field("id"),
		lp.Field("title"),
	).Columns(
		"COALESCE(meta->>'status', 'draft') AS status",
	).
		FromTable(users).
		LeftJoinLateral(latestPost, "lp", RawCondition("TRUE", nil)).
		Where(AllOf(
			Compare(Func("cardinality", Array(1, 2, 3)), "=", 3),
			Compare(users.Field("role"), "=", Any(Array("admin", "editor"))),
		))

	query, args := b.ToSQL()
	expected := "SELECT \"u\".\"id\", \"lp\".\"title\", COALESCE(meta->>'status', 'draft') AS status FROM \"users\" AS \"u\" LEFT JOIN LATERAL (SELECT \"p\".\"user_id\", \"p\".\"title\", ROW_NUMBER() OVER (ORDER BY \"p\".\"created_at\" DESC) AS rn FROM \"posts\" AS \"p\" WHERE \"p\".\"user_id\" = \"u\".\"id\" LIMIT 1) AS lp ON TRUE WHERE (cardinality(ARRAY[@array_0, @array_1, @array_2]) = @right AND \"u\".\"role\" = ANY(ARRAY[@array_0_1, @array_1_1]))"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 6 || args["array_0"] != 1 || args["array_1"] != 2 || args["array_2"] != 3 || args["right"] != 3 || args["array_0_1"] != "admin" || args["array_1_1"] != "editor" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestInsertFromSelect(t *testing.T) {
	users := TableOf("users")
	source := SelectFields(Field("id"), Field("email")).FromTable(users).Where(FieldEq(Field("confirmed"), true))

	b := Insert(nil).
		IntoTable(TableOf("newsletter_subscribers")).
		FromSelect(source, "user_id", "email").
		OnConflictDoNothing("user_id").
		Returning("user_id")

	query, args := b.ToSQL()
	expected := "INSERT INTO \"newsletter_subscribers\" (user_id, email) SELECT \"id\", \"email\" FROM \"users\" WHERE \"confirmed\" = @right ON CONFLICT (user_id) DO NOTHING RETURNING user_id"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 1 || args["right"] != true {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestTypedIdentifiersAndJSONHelpers(t *testing.T) {
	users := TableOf("public", "users").As("u")
	posts := TableAlias("posts", "p")

	b := SelectFields(
		users.Field("id"),
		users.Field("email"),
		posts.Field("title").As("post_title"),
	).
		FromTable(users).
		LeftJoinTable(posts, Compare(posts.Field("user_id"), "=", users.Field("id"))).
		Where(AllOf(
			JSONHasKey(users.Field("profile"), "status"),
			Compare(JSONGetText(users.Field("profile"), "status"), "=", "active"),
			ArrayOverlap(users.Field("roles"), Array("admin", "editor")),
			JSONHasAnyKeys(users.Field("settings"), "beta", "dark_mode"),
		)).
		GroupByFields(users.Field("id"), users.Field("email"), posts.Field("title"))

	query, args := b.ToSQL()
	expected := "SELECT \"u\".\"id\", \"u\".\"email\", \"p\".\"title\" AS \"post_title\" FROM \"public\".\"users\" AS \"u\" LEFT JOIN \"posts\" AS \"p\" ON \"p\".\"user_id\" = \"u\".\"id\" WHERE (\"u\".\"profile\" ? @right AND \"u\".\"profile\" ->> @right_1 = @right_2 AND \"u\".\"roles\" && ARRAY[@array_0, @array_1] AND \"u\".\"settings\" ?| ARRAY[@array_0_1, @array_1_1]) GROUP BY \"u\".\"id\", \"u\".\"email\", \"p\".\"title\""
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 7 || args["right"] != "status" || args["right_1"] != "status" || args["right_2"] != "active" || args["array_0"] != "admin" || args["array_1"] != "editor" || args["array_0_1"] != "beta" || args["array_1_1"] != "dark_mode" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestReturningFieldsAndDeleteKeyField(t *testing.T) {
	events := TableOf("audit", "events")
	b := DeleteFrom(events).
		DeleteKeyField(Field("audit", "events", "event_id")).
		Where(Compare(Field("audit", "events", "kind"), "=", "login")).
		OrderBy(Field("audit", "events", "created_at").Desc()).
		Limit(5).
		ReturningFields(Field("audit", "events", "event_id"))

	query, args := b.ToSQL()
	expected := "DELETE FROM \"audit\".\"events\" WHERE \"audit\".\"events\".\"event_id\" IN (SELECT \"audit\".\"events\".\"event_id\" FROM \"audit\".\"events\" WHERE \"audit\".\"events\".\"kind\" = @right ORDER BY \"audit\".\"events\".\"created_at\" DESC LIMIT 5) RETURNING \"audit\".\"events\".\"event_id\""
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 1 || args["right"] != "login" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestMatchFieldsBuildsTypedColumnPredicates(t *testing.T) {
	users := TableAlias("users", "u")
	cond := MatchFields(users, H{
		"email":  "dev@example.com",
		"status": "active",
	})

	query, args := SelectFields(users.AllFields()).FromTable(users).Where(cond).ToSQL()
	expected := "SELECT \"u\".* FROM \"users\" AS \"u\" WHERE (\"u\".\"email\" = @right AND \"u\".\"status\" = @right_1)"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 2 || args["right"] != "dev@example.com" || args["right_1"] != "active" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestLegacyRefAliasesRemainSupported(t *testing.T) {
	users := TableAlias("users", "u")

	query, args := SelectRefs(users.All(), users.Col("email")).
		FromTable(users).
		Where(EqRef(users.Col("enabled"), true)).
		OrderBy(users.Col("id").Desc()).
		Limit(1).
		ToSQL()

	expected := "SELECT \"u\".*, \"u\".\"email\" FROM \"users\" AS \"u\" WHERE \"u\".\"enabled\" = @right ORDER BY \"u\".\"id\" DESC LIMIT 1"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 1 || args["right"] != true {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestReadableTableHelpers(t *testing.T) {
	users := TableOf("public", "users").As("u")

	query, args := SelectFields(users.AllFields(), users.Field("email")).
		FromTable(users).
		Where(FieldEq(users.Field("enabled"), true)).
		ToSQL()

	expected := "SELECT \"u\".*, \"u\".\"email\" FROM \"public\".\"users\" AS \"u\" WHERE \"u\".\"enabled\" = @right"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 1 || args["right"] != true {
		t.Fatalf("unexpected args: %#v", args)
	}
}

type exampleTable struct{}

func (exampleTable) TableName() string {
	return "users"
}

func TestTableForAndFieldForHelpers(t *testing.T) {
	table := TableFor(exampleTable{}).As("u")

	query, args := SelectFields(table.AllFields()).
		FromTable(table).
		Where(FieldEq(FieldFor(exampleTable{}, "id"), 1)).
		ToSQL()

	expected := "SELECT \"u\".* FROM \"users\" AS \"u\" WHERE \"users\".\"id\" = @right"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 1 || args["right"] != 1 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestTableNameAndFieldNameAliases(t *testing.T) {
	var table TableName = TableAlias("users", "u")
	var field FieldName = table.Field("email")

	query, args := SelectFields(field).
		FromTable(table).
		Where(FieldLike(field, "%@example.com")).
		ToSQL()

	expected := "SELECT \"u\".\"email\" FROM \"users\" AS \"u\" WHERE \"u\".\"email\" LIKE @right"
	if query != expected {
		t.Fatalf("expected SQL %q, got %q", expected, query)
	}

	if len(args) != 1 || args["right"] != "%@example.com" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
