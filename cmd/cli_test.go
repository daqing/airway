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

	openapiPath := filepath.Join(wd, "app", "api", "admin_api", "openapi.go")
	openapiContent := readFile(t, openapiPath)
	if !strings.Contains(openapiContent, `openapi.Get("/admin/index"`) {
		t.Fatalf("expected generated openapi declaration, got:\n%s", openapiContent)
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

func TestGenerateMigrationCreatesUpAndDownFiles(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "db", "migrate"))

	frozenTime := time.Date(2026, 3, 27, 12, 34, 56, 0, time.UTC)
	previousNow := timeNow
	timeNow = func() time.Time { return frozenTime }
	t.Cleanup(func() {
		timeNow = previousNow
	})

	if err := run([]string{"generate", "migration", "create_posts"}); err != nil {
		t.Fatalf("run generate migration: %v", err)
	}

	upPath := filepath.Join(wd, "db", "migrate", "20260327123456_create_posts.up.sql")
	upContent := readFile(t, upPath)
	if !strings.Contains(upContent, "CREATE TABLE posts") {
		t.Fatalf("expected CREATE TABLE example in up migration, got:\n%s", upContent)
	}

	downPath := filepath.Join(wd, "db", "migrate", "20260327123456_create_posts.down.sql")
	downContent := readFile(t, downPath)
	if !strings.Contains(downContent, "DROP TABLE posts") {
		t.Fatalf("expected DROP TABLE example in down migration, got:\n%s", downContent)
	}
}

func TestCLIGenerateHelpPrintsUsage(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"cli", "generate", "-h"}); err != nil {
			t.Fatalf("run generate help: %v", err)
		}
	})

	if !strings.Contains(output, "airway generate [action|api|model|migration|service|cmd|island|scaffold] [params]") {
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
			expected: "airway generate action [api] [action]",
		},
		{
			name:     "api",
			args:     []string{"cli", "generate", "api", "-h"},
			expected: "airway generate api [name]",
		},
		{
			name:     "model",
			args:     []string{"cli", "generate", "model", "-h"},
			expected: "airway generate model [name] [field:type]...",
		},
		{
			name:     "migration",
			args:     []string{"cli", "generate", "migration", "-h"},
			expected: "airway generate migration [name]",
		},
		{
			name:     "service",
			args:     []string{"cli", "generate", "service", "-h"},
			expected: "airway generate service <name> <field:type> <field:type>...",
		},
		{
			name:     "cmd",
			args:     []string{"cli", "generate", "cmd", "-h"},
			expected: "airway generate cmd <name> <field> <field>...",
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

	if !strings.Contains(output, "airway generate migration [name]") {
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

func TestCLISchemaDumpUsesCurrentDatabaseSchema(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "tmp"))
	makeDirs(t, filepath.Join(wd, "db"))

	t.Setenv("AIRWAY_DB_DSN", "sqlite://./tmp/live-schema.sqlite")
	t.Setenv("AIRWAY_PG", "")

	db, err := repo.NewDB("sqlite://./tmp/live-schema.sqlite")
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
