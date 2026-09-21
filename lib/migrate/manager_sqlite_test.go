package migrate_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/migrate/schema"

	_ "modernc.org/sqlite"
)

// The migration engine historically lived in the CLI; these tests keep its
// SQLite behavior coverage (DSL migrations, SQLite table rebuilds, snapshots)
// against the exported API. They chdir into a temp dir and use relative
// paths, mirroring how projects run `airway db:migrate`.

func useTempWorkingDir(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	return tempDir
}

func makeDirs(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func managerOptions(t *testing.T, dsn string, out io.Writer) migrate.Options {
	t.Helper()

	return migrate.Options{
		DSN:          dsn,
		Migrations:   os.DirFS("db/migrate"),
		SnapshotPath: "db/schema.json",
		Out:          out,
	}
}

func openSQLite(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", strings.TrimPrefix(dsn, "sqlite://"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	return db
}

func readSnapshotFile(t *testing.T, path string) schema.Snapshot {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot file %s: %v", path, err)
	}

	var snapshot schema.Snapshot
	if err := json.Unmarshal(content, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot file %s: %v", path, err)
	}

	if snapshot.State == nil {
		snapshot.State = schema.NewState()
	}

	return snapshot
}

func TestMigrationManagerSupportsSQLiteMigrateStatusAndRollback(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))

	writeFile(t, filepath.Join(wd, "db", "migrate", "20260327120000_create_posts.up.sql"), `
CREATE TABLE posts (
  id INTEGER PRIMARY KEY,
  title TEXT NOT NULL
);
INSERT INTO posts (id, title) VALUES (1, 'hello');
`)
	writeFile(t, filepath.Join(wd, "db", "migrate", "20260327120000_create_posts.down.sql"), `
DROP TABLE posts;
`)

	makeDirs(t, filepath.Join(wd, "tmp"))
	dsn := "sqlite://./tmp/test.sqlite"
	opts := managerOptions(t, dsn, io.Discard)

	if err := migrate.Run(opts); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	db := openSQLite(t, dsn)
	defer db.Close()

	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM posts").Scan(&count); err != nil {
		t.Fatalf("count posts after migrate: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 post after migrate, got %d", count)
	}

	var statusOutput strings.Builder
	if err := migrate.Status(managerOptions(t, dsn, &statusOutput)); err != nil {
		t.Fatalf("status: %v", err)
	}

	if !strings.Contains(statusOutput.String(), "applied\t20260327120000_create_posts.up.sql") {
		t.Fatalf("expected applied migration in status output, got:\n%s", statusOutput.String())
	}

	if err := migrate.Rollback(opts, 1); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count schema_migrations after rollback: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected no applied migrations after rollback, got %d", count)
	}

	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM posts").Scan(&count); err == nil {
		t.Fatal("expected posts table to be removed after rollback")
	}
}

func TestMigrationManagerSupportsDSLMigrationOnSQLite(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))
	makeDirs(t, filepath.Join(wd, "tmp"))

	schema.ResetRegistryForTest()
	t.Cleanup(schema.ResetRegistryForTest)

	schema.RegisterChange("20260327130000", "create_widgets", func(m *schema.Migrator) {
		m.CreateTable("widgets", func(t *schema.Table) {
			t.ID()
			t.String("name", 100).Null(false)
			t.Boolean("enabled").Null(false).Default(true)
			t.Timestamps()
			t.UniqueIndex("name")
		})
	})

	dsn := "sqlite://./tmp/dsl.sqlite"
	opts := managerOptions(t, dsn, io.Discard)

	if err := migrate.Run(opts); err != nil {
		t.Fatalf("migrate dsl: %v", err)
	}

	db := openSQLite(t, dsn)
	defer db.Close()

	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", "20260327130000").Scan(&count); err != nil {
		t.Fatalf("check schema_migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected DSL migration version to be applied, got count %d", count)
	}

	if _, err := db.Exec(`INSERT INTO widgets (id, name, enabled, created_at, updated_at) VALUES (1, 'one', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert widget: %v", err)
	}

	if err := migrate.Rollback(opts, 1); err != nil {
		t.Fatalf("rollback dsl migration: %v", err)
	}

	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM widgets").Scan(&count); err == nil {
		t.Fatal("expected widgets table to be removed after DSL rollback")
	}
}

func TestMigrationManagerSupportsReferencesAndForeignKeysOnSQLite(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))
	makeDirs(t, filepath.Join(wd, "tmp"))

	schema.ResetRegistryForTest()
	t.Cleanup(schema.ResetRegistryForTest)

	schema.RegisterChange("20260327131000", "create_accounts", func(m *schema.Migrator) {
		m.CreateTable("accounts", func(t *schema.Table) {
			t.ID()
			t.String("name", 100).Null(false)
		})
	})

	schema.RegisterChange("20260327132000", "create_users", func(m *schema.Migrator) {
		m.CreateTable("users", func(t *schema.Table) {
			t.ID()
			t.String("email", 255).Null(false)
			t.References("account").Null(false).Index().ForeignKey().OnDelete("cascade")
			t.Timestamps()
		})
	})

	dsn := "sqlite://./tmp/refs.sqlite"
	opts := managerOptions(t, dsn, io.Discard)

	if err := migrate.Run(opts); err != nil {
		t.Fatalf("migrate references DSL: %v", err)
	}

	db := openSQLite(t, dsn)
	defer db.Close()

	var createSQL string
	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'users'`).Scan(&createSQL); err != nil {
		t.Fatalf("read users table sql: %v", err)
	}

	if !strings.Contains(createSQL, `FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE`) {
		t.Fatalf("expected foreign key in sqlite schema, got:\n%s", createSQL)
	}

	var indexSQL string
	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'idx_users_account_id'`).Scan(&indexSQL); err != nil {
		t.Fatalf("read users index sql: %v", err)
	}

	if !strings.Contains(indexSQL, `CREATE INDEX "idx_users_account_id" ON "users" ("account_id")`) {
		t.Fatalf("expected account index in sqlite schema, got:\n%s", indexSQL)
	}
}

func TestMigrationManagerSupportsRenameAndStandaloneIndexesOnSQLite(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))
	makeDirs(t, filepath.Join(wd, "tmp"))

	schema.ResetRegistryForTest()
	t.Cleanup(schema.ResetRegistryForTest)

	schema.RegisterChange("20260327133000", "create_users", func(m *schema.Migrator) {
		m.CreateTable("users", func(t *schema.Table) {
			t.ID()
			t.String("email", 255).Null(false)
		})
		m.AddIndex("users", "email").Unique().Name("users_email_unique_idx")
	})

	schema.RegisterChange("20260327134000", "rename_users", func(m *schema.Migrator) {
		m.RenameTable("users", "members")
		m.RenameColumn("members", "email", "login_email")
		m.RemoveIndex("members", "users_email_unique_idx")
		m.AddIndex("members", "login_email").Unique().Name("members_login_email_unique_idx")
	})

	dsn := "sqlite://./tmp/rename.sqlite"
	opts := managerOptions(t, dsn, io.Discard)

	if err := migrate.Run(opts); err != nil {
		t.Fatalf("migrate rename DSL: %v", err)
	}

	db := openSQLite(t, dsn)
	defer db.Close()

	var createSQL string
	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'members'`).Scan(&createSQL); err != nil {
		t.Fatalf("read members table sql: %v", err)
	}
	if !strings.Contains(createSQL, `"login_email" TEXT NOT NULL`) {
		t.Fatalf("expected renamed column in sqlite schema, got:\n%s", createSQL)
	}

	var indexCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'members_login_email_unique_idx'`).Scan(&indexCount); err != nil {
		t.Fatalf("read renamed index count: %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("expected renamed unique index to exist, got count %d", indexCount)
	}
}

func TestMigrationManagerSupportsSQLiteRemoveColumnViaRebuild(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))
	makeDirs(t, filepath.Join(wd, "tmp"))

	schema.ResetRegistryForTest()
	t.Cleanup(schema.ResetRegistryForTest)

	schema.RegisterChange("20260327135000", "create_profiles", func(m *schema.Migrator) {
		m.CreateTable("profiles", func(t *schema.Table) {
			t.ID()
			t.String("email", 255).Null(false)
			t.String("nickname", 100)
			t.Timestamps()
		})
	})

	schema.Register("20260327136000", "remove_nickname",
		func(m *schema.Migrator) {
			m.RemoveColumn("profiles", "nickname")
		},
		func(m *schema.Migrator) {
			m.AddColumn("profiles", schema.Column{
				Name: "nickname",
				Type: schema.Type{Kind: schema.TypeString, Length: 100},
			})
		},
	)

	dsn := "sqlite://./tmp/remove-column.sqlite"
	opts := managerOptions(t, dsn, io.Discard)

	if err := migrate.Run(opts); err != nil {
		t.Fatalf("migrate remove column DSL: %v", err)
	}

	db := openSQLite(t, dsn)
	defer db.Close()

	var createSQL string
	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'profiles'`).Scan(&createSQL); err != nil {
		t.Fatalf("read profiles table sql: %v", err)
	}
	if strings.Contains(createSQL, `"nickname"`) {
		t.Fatalf("expected nickname column to be removed, got:\n%s", createSQL)
	}

	if err := migrate.Rollback(opts, 1); err != nil {
		t.Fatalf("rollback remove column DSL: %v", err)
	}

	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'profiles'`).Scan(&createSQL); err != nil {
		t.Fatalf("read profiles table sql after rollback: %v", err)
	}
	if !strings.Contains(createSQL, `"nickname" TEXT`) {
		t.Fatalf("expected nickname column to be restored after rollback, got:\n%s", createSQL)
	}
}

func TestMigrationManagerSupportsSQLiteAddAndRemoveForeignKeyViaRebuild(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))
	makeDirs(t, filepath.Join(wd, "tmp"))

	schema.ResetRegistryForTest()
	t.Cleanup(schema.ResetRegistryForTest)

	schema.RegisterChange("20260327137000", "create_accounts", func(m *schema.Migrator) {
		m.CreateTable("accounts", func(t *schema.Table) {
			t.ID()
			t.String("name", 100).Null(false)
		})
	})

	schema.RegisterChange("20260327138000", "create_users", func(m *schema.Migrator) {
		m.CreateTable("users", func(t *schema.Table) {
			t.ID()
			t.BigInt("account_id").Null(false)
			t.String("email", 255).Null(false)
		})
		m.AddIndex("users", "account_id").Name("users_account_id_idx")
	})

	schema.Register("20260327139000", "add_users_account_fk",
		func(m *schema.Migrator) {
			m.AddForeignKey("users", "account_id", "accounts").Name("users_account_fk").OnDelete("cascade")
		},
		func(m *schema.Migrator) {
			m.RemoveForeignKey("users", "account_id", "accounts", "id")
		},
	)

	dsn := "sqlite://./tmp/fk-rebuild.sqlite"
	opts := managerOptions(t, dsn, io.Discard)

	if err := migrate.Run(opts); err != nil {
		t.Fatalf("migrate add foreign key DSL: %v", err)
	}

	db := openSQLite(t, dsn)
	defer db.Close()

	var createSQL string
	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'users'`).Scan(&createSQL); err != nil {
		t.Fatalf("read users table sql after fk add: %v", err)
	}
	if !strings.Contains(createSQL, `CONSTRAINT "users_account_fk" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE`) &&
		!strings.Contains(createSQL, `FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE`) {
		t.Fatalf("expected foreign key after rebuild, got:\n%s", createSQL)
	}

	if err := migrate.Rollback(opts, 1); err != nil {
		t.Fatalf("rollback add foreign key DSL: %v", err)
	}

	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'users'`).Scan(&createSQL); err != nil {
		t.Fatalf("read users table sql after fk rollback: %v", err)
	}
	if strings.Contains(createSQL, `FOREIGN KEY ("account_id") REFERENCES "accounts" ("id")`) {
		t.Fatalf("expected foreign key to be removed after rollback, got:\n%s", createSQL)
	}
}

func TestMigrationManagerSupportsSQLiteSetNullAndDefaultViaRebuild(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))
	makeDirs(t, filepath.Join(wd, "tmp"))

	schema.ResetRegistryForTest()
	t.Cleanup(schema.ResetRegistryForTest)

	schema.RegisterChange("20260327140000", "create_settings", func(m *schema.Migrator) {
		m.CreateTable("settings", func(t *schema.Table) {
			t.ID()
			t.String("name", 120)
			t.Boolean("enabled")
		})
	})

	schema.Register("20260327141000", "tighten_settings",
		func(m *schema.Migrator) {
			m.SetNull("settings", "name", false)
			m.SetDefault("settings", "enabled", true)
		},
		func(m *schema.Migrator) {
			m.SetNull("settings", "name", true)
			m.RemoveDefault("settings", "enabled")
		},
	)

	dsn := "sqlite://./tmp/set-null-default.sqlite"
	opts := managerOptions(t, dsn, io.Discard)

	if err := migrate.Run(opts); err != nil {
		t.Fatalf("migrate set null/default DSL: %v", err)
	}

	db := openSQLite(t, dsn)
	defer db.Close()

	var createSQL string
	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'settings'`).Scan(&createSQL); err != nil {
		t.Fatalf("read settings table sql: %v", err)
	}
	if !strings.Contains(createSQL, `"name" TEXT NOT NULL`) {
		t.Fatalf("expected name NOT NULL after rebuild, got:\n%s", createSQL)
	}
	if !strings.Contains(createSQL, `"enabled" INTEGER DEFAULT TRUE`) {
		t.Fatalf("expected enabled default after rebuild, got:\n%s", createSQL)
	}

	if err := migrate.Rollback(opts, 1); err != nil {
		t.Fatalf("rollback set null/default DSL: %v", err)
	}

	if err := db.QueryRowContext(context.Background(), `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'settings'`).Scan(&createSQL); err != nil {
		t.Fatalf("read settings table sql after rollback: %v", err)
	}
	if strings.Contains(createSQL, `"name" TEXT NOT NULL`) {
		t.Fatalf("expected name nullability to be restored after rollback, got:\n%s", createSQL)
	}
	if strings.Contains(createSQL, `"enabled" INTEGER DEFAULT TRUE`) {
		t.Fatalf("expected enabled default to be removed after rollback, got:\n%s", createSQL)
	}
}

func TestMigrationManagerWritesSchemaSnapshotAfterDSLMigrateAndRollback(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))
	makeDirs(t, filepath.Join(wd, "tmp"))

	schema.ResetRegistryForTest()
	t.Cleanup(schema.ResetRegistryForTest)

	schema.RegisterChange("20260327142000", "create_projects", func(m *schema.Migrator) {
		m.CreateTable("projects", func(t *schema.Table) {
			t.ID()
			t.String("name", 120).Null(false)
		})
	})

	dsn := "sqlite://./tmp/snapshot.sqlite"
	opts := managerOptions(t, dsn, io.Discard)

	if err := migrate.Run(opts); err != nil {
		t.Fatalf("migrate snapshot DSL: %v", err)
	}

	snapshot := readSnapshotFile(t, filepath.Join(wd, "db", "schema.json"))
	if !snapshot.Known {
		t.Fatal("expected schema snapshot to be marked known after DSL migrate")
	}
	if _, ok := snapshot.State.Tables["projects"]; !ok {
		t.Fatalf("expected projects table in schema snapshot, got %#v", snapshot.State.Tables)
	}

	if err := migrate.Rollback(opts, 1); err != nil {
		t.Fatalf("rollback snapshot DSL: %v", err)
	}

	snapshot = readSnapshotFile(t, filepath.Join(wd, "db", "schema.json"))
	if _, ok := snapshot.State.Tables["projects"]; ok {
		t.Fatalf("expected projects table to be removed from schema snapshot after rollback, got %#v", snapshot.State.Tables)
	}
}
