package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

	upPath := filepath.Join(wd, "db", "migrate", "20260327123456_create_posts.up.sql")
	downPath := filepath.Join(wd, "db", "migrate", "20260327123456_create_posts.down.sql")

	if _, err := os.Stat(upPath); err != nil {
		t.Fatalf("expected up migration file: %v", err)
	}

	if _, err := os.Stat(downPath); err != nil {
		t.Fatalf("expected down migration file: %v", err)
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
