package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
)

func TestRunCLICommandGeneratesModel(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "app", "models"))

	if err := run([]string{"cli", "generate", "model", "post"}); err != nil {
		t.Fatalf("run generate model: %v", err)
	}

	content := readFile(t, filepath.Join(wd, "app", "models", "post.go"))
	if !strings.Contains(content, "type Post struct") {
		t.Fatalf("expected generated model type, got:\n%s", content)
	}

	if !strings.Contains(content, `return "posts"`) {
		t.Fatalf("expected pluralized table name, got:\n%s", content)
	}

	if !strings.Contains(content, `registerREPLModel("Post", Post{})`) {
		t.Fatalf("expected REPL registration, got:\n%s", content)
	}
}

func TestRunCLICommandGeneratesAPI(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "app", "api"))

	if err := run([]string{"cli", "generate", "api", "admin"}); err != nil {
		t.Fatalf("run generate api: %v", err)
	}

	routesPath := filepath.Join(wd, "app", "api", "admin_api", "routes.go")
	routesContent := readFile(t, routesPath)
	if !strings.Contains(routesContent, `g := r.Group("/admin")`) {
		t.Fatalf("expected generated admin route group, got:\n%s", routesContent)
	}

	actionPath := filepath.Join(wd, "app", "api", "admin_api", "index_action.go")
	actionContent := readFile(t, actionPath)
	if !strings.Contains(actionContent, "func IndexAction") {
		t.Fatalf("expected IndexAction, got:\n%s", actionContent)
	}

	if !strings.Contains(actionContent, "render.Empty(c)") {
		t.Fatalf("expected render.Empty response, got:\n%s", actionContent)
	}
}

func TestRunCLICommandGeneratesServiceAndCmdTemplates(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "app", "services"))
	makeDirs(t, filepath.Join(wd, "cmd"))

	if err := run([]string{"cli", "generate", "service", "post", "title:string", "published:bool"}); err != nil {
		t.Fatalf("run generate service: %v", err)
	}

	serviceContent := readFile(t, filepath.Join(wd, "app", "services", "post.go"))
	if !strings.Contains(serviceContent, "func CreatePost(title string, published bool)") {
		t.Fatalf("expected generated service signature, got:\n%s", serviceContent)
	}

	if !strings.Contains(serviceContent, `sql.H{"title": title, "published": published}`) {
		t.Fatalf("expected generated insert hash, got:\n%s", serviceContent)
	}

	if err := run([]string{"cli", "generate", "cmd", "post", "title", "published"}); err != nil {
		t.Fatalf("run generate cmd: %v", err)
	}

	cmdContent := readFile(t, filepath.Join(wd, "cmd", "post.go"))
	if !strings.Contains(cmdContent, `if len(args) != 2`) {
		t.Fatalf("expected create arg validation, got:\n%s", cmdContent)
	}

	if !strings.Contains(cmdContent, `if len(args) != 3`) {
		t.Fatalf("expected update arg validation, got:\n%s", cmdContent)
	}
}

func TestCLIPluginInstallCopiesAppCmdAndMigrations(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "app", "api", "demo_api"))
	makeDirs(t, filepath.Join(wd, "cmd"))
	makeDirs(t, filepath.Join(wd, "db", "migrate"))

	writeFile(t, filepath.Join(wd, "app", "api", "demo_api", "routes.go"), "package demo_api\n")
	writeFile(t, filepath.Join(wd, "cmd", "demo.go"), "package cmd\n")
	writeFile(t, filepath.Join(wd, "db", "migrate", "create_demo.sql"), "-- demo\n")

	targetDir := filepath.Join(wd, "target")
	makeDirs(t, filepath.Join(targetDir, "app"))
	makeDirs(t, filepath.Join(targetDir, "cmd"))
	makeDirs(t, filepath.Join(targetDir, "db", "migrate"))

	frozenTime := time.Date(2026, 3, 27, 12, 34, 56, 0, time.UTC)
	previousNow := timeNow
	timeNow = func() time.Time { return frozenTime }
	t.Cleanup(func() {
		timeNow = previousNow
	})

	if err := run([]string{"cli", "plugin", "install", targetDir}); err != nil {
		t.Fatalf("run plugin install: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "app", "api", "demo_api", "routes.go")); err != nil {
		t.Fatalf("expected app files copied: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "cmd", "demo.go")); err != nil {
		t.Fatalf("expected cmd files copied: %v", err)
	}

	migrationPath := filepath.Join(targetDir, "db", "migrate", "20260327123456_create_demo.sql")
	if _, err := os.Stat(migrationPath); err != nil {
		t.Fatalf("expected timestamped migration copied: %v", err)
	}
}

func TestGenerateMigrationCreatesUpAndDownFiles(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))

	frozenTime := time.Date(2026, 3, 27, 12, 34, 56, 0, time.UTC)
	previousNow := timeNow
	timeNow = func() time.Time { return frozenTime }
	t.Cleanup(func() {
		timeNow = previousNow
	})

	if err := run([]string{"cli", "generate", "migration", "create_posts"}); err != nil {
		t.Fatalf("run generate migration: %v", err)
	}

	migrationPath := filepath.Join(wd, "db", "migrate", "20260327123456_create_posts.go")

	content := readFile(t, migrationPath)
	if !strings.Contains(content, `schema.RegisterChange("20260327123456", "create_posts"`) {
		t.Fatalf("expected DSL migration template, got:\n%s", content)
	}
}

func TestCLIGenerateHelpPrintsUsage(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"cli", "generate", "-h"}); err != nil {
			t.Fatalf("run generate help: %v", err)
		}
	})

	if !strings.Contains(output, "airway cli generate [action|api|model|migration|service|cmd] [params]") {
		t.Fatalf("expected generate usage output, got:\n%s", output)
	}
}

func TestCLIGenerateSubcommandHelpPrintsUsage(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "action",
			args:     []string{"cli", "generate", "action", "-h"},
			expected: "airway cli generate action [api] [action]",
		},
		{
			name:     "api",
			args:     []string{"cli", "generate", "api", "-h"},
			expected: "airway cli generate api [name]",
		},
		{
			name:     "model",
			args:     []string{"cli", "generate", "model", "-h"},
			expected: "airway cli generate model [name] [field:type]...",
		},
		{
			name:     "migration",
			args:     []string{"cli", "generate", "migration", "-h"},
			expected: "airway cli generate migration [name]",
		},
		{
			name:     "service",
			args:     []string{"cli", "generate", "service", "-h"},
			expected: "airway cli generate service <name> <field:type> <field:type>...",
		},
		{
			name:     "cmd",
			args:     []string{"cli", "generate", "cmd", "-h"},
			expected: "airway cli generate cmd <name> <field> <field>...",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := run(tc.args); err != nil {
					t.Fatalf("run help: %v", err)
				}
			})

			if !strings.Contains(output, tc.expected) {
				t.Fatalf("expected help output %q, got:\n%s", tc.expected, output)
			}
		})
	}
}

func TestGenerateMigrationHelpPrintsUsageWithoutCreatingFile(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))

	output := captureStdout(t, func() {
		if err := run([]string{"cli", "generate", "migration", "-h"}); err != nil {
			t.Fatalf("run generate migration help: %v", err)
		}
	})

	if !strings.Contains(output, "airway cli generate migration [name]") {
		t.Fatalf("expected migration usage output, got:\n%s", output)
	}

	entries, err := os.ReadDir(filepath.Join(wd, "db", "migrate"))
	if err != nil {
		t.Fatalf("read migration dir: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected no migration files for help, found %d", len(entries))
	}
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
	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/test.sqlite3")
	t.Setenv("AIRWAY_PG", "")

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		t.Fatalf("newMigrationManagerFromEnv: %v", err)
	}
	defer manager.Close()

	if err := manager.Migrate(""); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var count int
	if err := manager.db.Conn().Get(&count, "SELECT COUNT(*) FROM posts"); err != nil {
		t.Fatalf("count posts after migrate: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 post after migrate, got %d", count)
	}

	statusOutput := captureStdout(t, func() {
		if err := manager.Status(); err != nil {
			t.Fatalf("status: %v", err)
		}
	})

	if !strings.Contains(statusOutput, "applied\t20260327120000_create_posts.up.sql") {
		t.Fatalf("expected applied migration in status output, got:\n%s", statusOutput)
	}

	if err := manager.Rollback(1); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if err := manager.db.Conn().Get(&count, "SELECT COUNT(*) FROM schema_migrations"); err != nil {
		t.Fatalf("count schema_migrations after rollback: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected no applied migrations after rollback, got %d", count)
	}

	if err := manager.db.Conn().Get(&count, "SELECT COUNT(*) FROM posts"); err == nil {
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

	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/dsl.sqlite3")
	t.Setenv("AIRWAY_PG", "")

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		t.Fatalf("newMigrationManagerFromEnv: %v", err)
	}
	defer manager.Close()

	if err := manager.Migrate(""); err != nil {
		t.Fatalf("migrate dsl: %v", err)
	}

	var count int
	if err := manager.db.Conn().Get(&count, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", "20260327130000"); err != nil {
		t.Fatalf("check schema_migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected DSL migration version to be applied, got count %d", count)
	}

	if _, err := manager.db.Conn().Exec(`INSERT INTO widgets (id, name, enabled, created_at, updated_at) VALUES (1, 'one', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert widget: %v", err)
	}

	if err := manager.Rollback(1); err != nil {
		t.Fatalf("rollback dsl migration: %v", err)
	}

	if err := manager.db.Conn().Get(&count, "SELECT COUNT(*) FROM widgets"); err == nil {
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

	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/refs.sqlite3")
	t.Setenv("AIRWAY_PG", "")

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		t.Fatalf("newMigrationManagerFromEnv: %v", err)
	}
	defer manager.Close()

	if err := manager.Migrate(""); err != nil {
		t.Fatalf("migrate references DSL: %v", err)
	}

	var createSQL string
	if err := manager.db.Conn().Get(&createSQL, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'users'`); err != nil {
		t.Fatalf("read users table sql: %v", err)
	}

	if !strings.Contains(createSQL, `FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE`) {
		t.Fatalf("expected foreign key in sqlite schema, got:\n%s", createSQL)
	}

	var indexSQL string
	if err := manager.db.Conn().Get(&indexSQL, `SELECT sql FROM sqlite_master WHERE type = 'index' AND name = 'idx_users_account_id'`); err != nil {
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

	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/rename.sqlite3")
	t.Setenv("AIRWAY_PG", "")

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		t.Fatalf("newMigrationManagerFromEnv: %v", err)
	}
	defer manager.Close()

	if err := manager.Migrate(""); err != nil {
		t.Fatalf("migrate rename DSL: %v", err)
	}

	var createSQL string
	if err := manager.db.Conn().Get(&createSQL, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'members'`); err != nil {
		t.Fatalf("read members table sql: %v", err)
	}
	if !strings.Contains(createSQL, `"login_email" TEXT NOT NULL`) {
		t.Fatalf("expected renamed column in sqlite schema, got:\n%s", createSQL)
	}

	var indexCount int
	if err := manager.db.Conn().Get(&indexCount, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'members_login_email_unique_idx'`); err != nil {
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

	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/remove-column.sqlite3")
	t.Setenv("AIRWAY_PG", "")

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		t.Fatalf("newMigrationManagerFromEnv: %v", err)
	}
	defer manager.Close()

	if err := manager.Migrate(""); err != nil {
		t.Fatalf("migrate remove column DSL: %v", err)
	}

	var createSQL string
	if err := manager.db.Conn().Get(&createSQL, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'profiles'`); err != nil {
		t.Fatalf("read profiles table sql: %v", err)
	}
	if strings.Contains(createSQL, `"nickname"`) {
		t.Fatalf("expected nickname column to be removed, got:\n%s", createSQL)
	}

	if err := manager.Rollback(1); err != nil {
		t.Fatalf("rollback remove column DSL: %v", err)
	}

	if err := manager.db.Conn().Get(&createSQL, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'profiles'`); err != nil {
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

	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/fk-rebuild.sqlite3")
	t.Setenv("AIRWAY_PG", "")

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		t.Fatalf("newMigrationManagerFromEnv: %v", err)
	}
	defer manager.Close()

	if err := manager.Migrate(""); err != nil {
		t.Fatalf("migrate add foreign key DSL: %v", err)
	}

	var createSQL string
	if err := manager.db.Conn().Get(&createSQL, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'users'`); err != nil {
		t.Fatalf("read users table sql after fk add: %v", err)
	}
	if !strings.Contains(createSQL, `CONSTRAINT "users_account_fk" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE`) &&
		!strings.Contains(createSQL, `FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE`) {
		t.Fatalf("expected foreign key after rebuild, got:\n%s", createSQL)
	}

	if err := manager.Rollback(1); err != nil {
		t.Fatalf("rollback add foreign key DSL: %v", err)
	}

	if err := manager.db.Conn().Get(&createSQL, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'users'`); err != nil {
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

	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/set-null-default.sqlite3")
	t.Setenv("AIRWAY_PG", "")

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		t.Fatalf("newMigrationManagerFromEnv: %v", err)
	}
	defer manager.Close()

	if err := manager.Migrate(""); err != nil {
		t.Fatalf("migrate set null/default DSL: %v", err)
	}

	var createSQL string
	if err := manager.db.Conn().Get(&createSQL, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'settings'`); err != nil {
		t.Fatalf("read settings table sql: %v", err)
	}
	if !strings.Contains(createSQL, `"name" TEXT NOT NULL`) {
		t.Fatalf("expected name NOT NULL after rebuild, got:\n%s", createSQL)
	}
	if !strings.Contains(createSQL, `"enabled" INTEGER DEFAULT TRUE`) {
		t.Fatalf("expected enabled default after rebuild, got:\n%s", createSQL)
	}

	if err := manager.Rollback(1); err != nil {
		t.Fatalf("rollback set null/default DSL: %v", err)
	}

	if err := manager.db.Conn().Get(&createSQL, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'settings'`); err != nil {
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

	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/snapshot.sqlite3")
	t.Setenv("AIRWAY_PG", "")

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		t.Fatalf("newMigrationManagerFromEnv: %v", err)
	}
	defer manager.Close()

	if err := manager.Migrate(""); err != nil {
		t.Fatalf("migrate snapshot DSL: %v", err)
	}

	snapshot := readSnapshotFile(t, filepath.Join(wd, "db", "schema.json"))
	if !snapshot.Known {
		t.Fatal("expected schema snapshot to be marked known after DSL migrate")
	}
	if _, ok := snapshot.State.Tables["projects"]; !ok {
		t.Fatalf("expected projects table in schema snapshot, got %#v", snapshot.State.Tables)
	}

	if err := manager.Rollback(1); err != nil {
		t.Fatalf("rollback snapshot DSL: %v", err)
	}

	snapshot = readSnapshotFile(t, filepath.Join(wd, "db", "schema.json"))
	if _, ok := snapshot.State.Tables["projects"]; ok {
		t.Fatalf("expected projects table to be removed from schema snapshot after rollback, got %#v", snapshot.State.Tables)
	}
}

func TestCLISchemaDumpUsesCurrentDatabaseSchema(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "tmp"))
	makeDirs(t, filepath.Join(wd, "db"))

	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/live-schema.sqlite3")
	t.Setenv("AIRWAY_PG", "")

	db, err := repo.NewDB("sqlite://./tmp/live-schema.sqlite3")
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	defer db.Close()

	if _, err := db.Conn().Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	if _, err := db.Conn().Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL UNIQUE
);
`); err != nil {
		t.Fatalf("create users table: %v", err)
	}

	if _, err := db.Conn().Exec(`
CREATE TABLE audit_logs (
  id INTEGER PRIMARY KEY,
  user_id INTEGER NOT NULL,
  payload TEXT,
  FOREIGN KEY(user_id) REFERENCES users(id)
);
`); err != nil {
		t.Fatalf("create audit_logs table: %v", err)
	}

	if _, err := db.Conn().Exec(`CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id)`); err != nil {
		t.Fatalf("create index: %v", err)
	}

	if err := schema.SaveSnapshot(filepath.Join(wd, "db", "schema.json"), &schema.State{
		Tables: map[string]*schema.TableState{
			"stale_table": {Name: "stale_table"},
		},
	}, true); err != nil {
		t.Fatalf("write stale snapshot: %v", err)
	}

	if err := runCLISchemaDump(nil); err != nil {
		t.Fatalf("schema dump: %v", err)
	}

	snapshot := readSnapshotFile(t, filepath.Join(wd, "db", "schema.json"))
	if !snapshot.Known {
		t.Fatal("expected schema snapshot to be marked known")
	}

	if _, ok := snapshot.State.Tables["stale_table"]; ok {
		t.Fatalf("expected stale snapshot contents to be replaced, got %#v", snapshot.State.Tables)
	}

	users, ok := snapshot.State.Tables["users"]
	if !ok {
		t.Fatalf("expected users table in snapshot, got %#v", snapshot.State.Tables)
	}

	auditLogs, ok := snapshot.State.Tables["audit_logs"]
	if !ok {
		t.Fatalf("expected audit_logs table in snapshot, got %#v", snapshot.State.Tables)
	}

	emailColumn, found := snapshot.State.Column("users", "email")
	if !found {
		t.Fatalf("expected users.email column in snapshot, got %#v", users.Columns)
	}
	if emailColumn.Null == nil || *emailColumn.Null {
		t.Fatalf("expected users.email to be not null, got %#v", emailColumn)
	}

	if len(auditLogs.ForeignKeys) == 0 {
		t.Fatalf("expected audit_logs foreign keys in snapshot, got %#v", auditLogs.ForeignKeys)
	}
}

func TestCLIDSNPrefersCurrentProjectEnvNames(t *testing.T) {
	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/airway.db")
	t.Setenv("AIRWAY_PG", "postgres://legacy")

	dsn, err := cliDSN()
	if err != nil {
		t.Fatalf("cliDSN returned error: %v", err)
	}

	if dsn != "sqlite://./tmp/airway.db" {
		t.Fatalf("expected AIRWAY_DB_DSN, got %q", dsn)
	}
}

func TestCLIDSNFallsBackToLegacyEnvName(t *testing.T) {
	t.Setenv("AIRWAY_DB_DSN", "")
	t.Setenv("AIRWAY_PG", "postgres://legacy")

	dsn, err := cliDSN()
	if err != nil {
		t.Fatalf("cliDSN returned error: %v", err)
	}

	if dsn != "postgres://legacy" {
		t.Fatalf("expected AIRWAY_PG fallback, got %q", dsn)
	}
}

func TestGenerateModelReturnsExistsError(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "app", "models"))
	writeFile(t, filepath.Join(wd, "app", "models", "post.go"), "package models\n")

	err := run([]string{"cli", "generate", "model", "post"})
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected os.ErrExist, got %v", err)
	}
}

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

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}

	return string(content)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}

	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	_ = writer.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	return string(content)
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
