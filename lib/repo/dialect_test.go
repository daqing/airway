package repo

import (
	"reflect"
	"strings"
	"testing"

	buildersql "github.com/daqing/airway/lib/sql"
)

func TestCompileNamedQueryUsesOccurrenceOrder(t *testing.T) {
	query, args, err := compileNamedQuery("a = @a OR b = @b OR c = @a", buildersql.NamedArgs{"a": 1, "b": 2})
	if err != nil {
		t.Fatalf("compile named query: %v", err)
	}

	if query != "a = ? OR b = ? OR c = ?" {
		t.Fatalf("unexpected compiled query: %q", query)
	}

	expectedArgs := []any{1, 2, 1}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Fatalf("expected args %#v, got %#v", expectedArgs, args)
	}
}

func TestCompileNamedQueryFailsWhenArgMissing(t *testing.T) {
	_, _, err := compileNamedQuery("a = @a AND b = @missing", buildersql.NamedArgs{"a": 1})
	if err == nil {
		t.Fatal("expected missing arg error")
	}
}

func TestTransformQueryForDriverSQLiteRewritesILike(t *testing.T) {
	query := `SELECT "users".* FROM "users" WHERE "email" ILIKE @right`

	transformed, err := transformQueryForDriver(DriverSQLite, query)
	if err != nil {
		t.Fatalf("transform sqlite query: %v", err)
	}

	expected := `SELECT "users".* FROM "users" WHERE "email" LIKE @right`
	if transformed != expected {
		t.Fatalf("expected %q, got %q", expected, transformed)
	}
}

func TestPrepareBuilderStripsLockClausesForSQLite(t *testing.T) {
	jobs := buildersql.TableOf("jobs")

	cases := map[string]*buildersql.Builder{
		"for update":             buildersql.SelectFields(jobs.AllFields()).FromTable(jobs).ForUpdate(),
		"for share":              buildersql.SelectFields(jobs.AllFields()).FromTable(jobs).ForShare(),
		"for update skip locked": buildersql.SelectFields(jobs.AllFields()).FromTable(jobs).ForUpdateSkipLocked(),
		"for share skip locked":  buildersql.SelectFields(jobs.AllFields()).FromTable(jobs).ForShareSkipLocked(),
		"raw clause":             buildersql.SelectFields(jobs.AllFields()).FromTable(jobs).For("FOR UPDATE SKIP LOCKED"),
	}

	for name, stmt := range cases {
		query, args, err := DriverSQLite.prepareBuilder(stmt)
		if err != nil {
			t.Fatalf("%s: prepare builder: %v", name, err)
		}

		expected := `SELECT "jobs".* FROM "jobs"`
		if query != expected {
			t.Fatalf("%s: expected %q, got %q", name, expected, query)
		}

		if len(args) != 0 {
			t.Fatalf("%s: expected no args, got %#v", name, args)
		}
	}
}

func TestPrepareBuilderUnwrapsDeleteSubqueryForMySQL(t *testing.T) {
	todos := buildersql.TableOf("todos")
	stmt := buildersql.DeleteFrom(todos).OrderBy(todos.Field("id").Asc()).Limit(1)

	mysqlQuery, _, err := DriverMySQL.prepareBuilder(stmt)
	if err != nil {
		t.Fatalf("prepare mysql builder: %v", err)
	}

	expected := "DELETE FROM `todos` ORDER BY `todos`.`id` ASC LIMIT 1"
	if mysqlQuery != expected {
		t.Fatalf("expected %q, got %q", expected, mysqlQuery)
	}

	sqliteQuery, _, err := DriverSQLite.prepareBuilder(stmt)
	if err != nil {
		t.Fatalf("prepare sqlite builder: %v", err)
	}

	expected = `DELETE FROM "todos" WHERE id IN (SELECT id FROM "todos" ORDER BY "todos"."id" ASC LIMIT 1)`
	if sqliteQuery != expected {
		t.Fatalf("expected %q, got %q", expected, sqliteQuery)
	}
}

func TestTransformQueryForDriverSQLiteRejectsConstraintConflict(t *testing.T) {
	_, err := transformQueryForDriver(DriverSQLite, `INSERT INTO "users" ("name") VALUES (@name) ON CONFLICT ON CONSTRAINT users_name_key DO NOTHING`)
	if err == nil {
		t.Fatal("expected sqlite conflict-on-constraint error")
	}
}

func TestTransformMySQLQueryConvertsQuotesAndConflict(t *testing.T) {
	query := `INSERT INTO "users" ("id", "name") VALUES (@id, @name) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`

	transformed, err := transformMySQLQuery(query)
	if err != nil {
		t.Fatalf("transform query: %v", err)
	}

	expected := "INSERT INTO `users` (`id`, `name`) VALUES (@id, @name) ON DUPLICATE KEY UPDATE name = VALUES(name)"
	if transformed != expected {
		t.Fatalf("expected %q, got %q", expected, transformed)
	}
}

func TestTransformMySQLQueryConvertsNotILike(t *testing.T) {
	query := `SELECT "users".* FROM "users" WHERE "email" NOT ILIKE @right`

	transformed, err := transformMySQLQuery(query)
	if err != nil {
		t.Fatalf("transform query: %v", err)
	}

	expected := "SELECT `users`.* FROM `users` WHERE `email` NOT LIKE @right"
	if transformed != expected {
		t.Fatalf("expected %q, got %q", expected, transformed)
	}
}

func TestTransformMySQLQueryRejectsReturning(t *testing.T) {
	_, err := transformMySQLQuery(`INSERT INTO "users" ("name") VALUES (@name) RETURNING *`)
	if err == nil {
		t.Fatal("expected RETURNING to be rejected for mysql")
	}
}

func TestTransformMySQLQueryConvertsDoNothingToInsertIgnore(t *testing.T) {
	query := `INSERT INTO "users" ("email") VALUES (@email) ON CONFLICT (email) DO NOTHING`

	transformed, err := transformMySQLQuery(query)
	if err != nil {
		t.Fatalf("transform query: %v", err)
	}

	expected := "INSERT IGNORE INTO `users` (`email`) VALUES (@email)"
	if transformed != expected {
		t.Fatalf("expected %q, got %q", expected, transformed)
	}
}

func TestStripReturningClause(t *testing.T) {
	query := `INSERT INTO "users" ("title") VALUES (@title) RETURNING *`
	stripped := stripReturningClause(query)
	expected := `INSERT INTO "users" ("title") VALUES (@title)`
	if stripped != expected {
		t.Fatalf("expected %q, got %q", expected, stripped)
	}
}

func TestResolveDriverAutoDetectsSQLiteAndMySQL(t *testing.T) {
	mysqlDriver, mysqlDSN, err := resolveDriver("", "mysql://root:pass@127.0.0.1:3306/airway?charset=utf8mb4")
	if err != nil {
		t.Fatalf("resolve mysql driver: %v", err)
	}

	if mysqlDriver != DriverMySQL {
		t.Fatalf("expected mysql driver, got %q", mysqlDriver)
	}

	expectedMySQLDSN := "root:pass@tcp(127.0.0.1:3306)/airway?charset=utf8mb4&parseTime=true"
	if mysqlDSN != expectedMySQLDSN {
		t.Fatalf("expected mysql dsn %q, got %q", expectedMySQLDSN, mysqlDSN)
	}

	sqliteDriver, sqliteDSN, err := resolveDriver("", "sqlite://./tmp/test.db")
	if err != nil {
		t.Fatalf("resolve sqlite driver: %v", err)
	}

	if sqliteDriver != DriverSQLite {
		t.Fatalf("expected sqlite driver, got %q", sqliteDriver)
	}

	if sqliteDSN != "./tmp/test.db" {
		t.Fatalf("expected sqlite dsn %q, got %q", "./tmp/test.db", sqliteDSN)
	}
}

func TestNormalizeDriverSupportsMySQL84Alias(t *testing.T) {
	driver, err := normalizeDriver("mysql8.4")
	if err != nil {
		t.Fatalf("normalize mysql8.4 driver: %v", err)
	}

	if driver != DriverMySQL {
		t.Fatalf("expected mysql driver, got %q", driver)
	}
}

func TestNormalizeMySQLDSNPreservesParseTime(t *testing.T) {
	dsn := normalizeMySQLDSN("root:pass@tcp(127.0.0.1:3306)/airway?charset=utf8mb4&parseTime=true")
	if strings.Count(dsn, "parseTime=true") != 1 {
		t.Fatalf("expected parseTime to appear once, got %q", dsn)
	}
}
