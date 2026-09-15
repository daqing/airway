package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestInstallPluginMigrations(t *testing.T) {
	migrations := fstest.MapFS{
		"db/migrate/20240101000000_create_messages.up.sql": {
			Data: []byte("CREATE TABLE messages (id INTEGER PRIMARY KEY);"),
		},
		"db/migrate/20240101000000_create_messages.down.sql": {
			Data: []byte("DROP TABLE messages;"),
		},
		"db/migrate/20240102000000_add_body_to_messages.up.sql": {
			Data: []byte("ALTER TABLE messages ADD COLUMN body TEXT;"),
		},
		"db/migrate/20240102000000_add_body_to_messages.down.sql": {
			Data: []byte("ALTER TABLE messages DROP COLUMN body;"),
		},
		"README.md": {Data: []byte("not a migration")},
	}

	dstDir := filepath.Join(t.TempDir(), "db", "migrate")
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)

	if err := installPluginMigrations("im", migrations, dstDir, now); err != nil {
		t.Fatalf("install plugin migrations: %v", err)
	}

	up := readFile(t, filepath.Join(dstDir, "20260911120000_add_body_to_messages.up.sql"))
	if up != "ALTER TABLE messages ADD COLUMN body TEXT;" {
		t.Fatalf("unexpected up migration content: %s", up)
	}

	down := readFile(t, filepath.Join(dstDir, "20260911120001_create_messages.down.sql"))
	if down != "DROP TABLE messages;" {
		t.Fatalf("unexpected down migration content: %s", down)
	}

	// Installing again must skip instead of duplicating.
	if err := installPluginMigrations("im", migrations, dstDir, now); err != nil {
		t.Fatalf("reinstall plugin migrations: %v", err)
	}

	entries, err := os.ReadDir(dstDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected 4 migration files after reinstall, got %d", len(entries))
	}
}

func TestInstallPluginMigrationsRejectsMissingDown(t *testing.T) {
	migrations := fstest.MapFS{
		"20240101000000_create_messages.up.sql": {Data: []byte("SELECT 1;")},
	}

	dstDir := filepath.Join(t.TempDir(), "db", "migrate")

	if err := installPluginMigrations("im", migrations, dstDir, time.Now()); err == nil {
		t.Fatal("expected an error for a migration without a down file")
	}
}

func TestEnsurePluginImportAppendsImportBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugins.go")
	writeFile(t, path, "package main\n\n// Plugins are enabled via blank imports.\n")

	added, err := ensurePluginImport(path, "github.com/daqing/airway-im-plugin")
	if err != nil {
		t.Fatalf("ensure plugin import: %v", err)
	}
	if !added {
		t.Fatal("expected the file to change")
	}

	content := readFile(t, path)
	if !strings.Contains(content, "import (\n\t_ \"github.com/daqing/airway-im-plugin\"\n)") {
		t.Fatalf("expected appended import block, got:\n%s", content)
	}
}

func TestEnsurePluginImportReusesExistingBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugins.go")
	writeFile(t, path, "package main\n\nimport (\n\t_ \"github.com/example/airway-billing-plugin\"\n)\n")

	added, err := ensurePluginImport(path, "github.com/daqing/airway-im-plugin")
	if err != nil {
		t.Fatalf("ensure plugin import: %v", err)
	}
	if !added {
		t.Fatal("expected the file to change")
	}

	content := readFile(t, path)
	for _, want := range []string{`_ "github.com/example/airway-billing-plugin"`, `_ "github.com/daqing/airway-im-plugin"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %s in:\n%s", want, content)
		}
	}
}

func TestEnsurePluginImportIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugins.go")
	writeFile(t, path, "package main\n\nimport (\n\t_ \"github.com/daqing/airway-im-plugin\"\n)\n")

	added, err := ensurePluginImport(path, "github.com/daqing/airway-im-plugin")
	if err != nil {
		t.Fatalf("ensure plugin import: %v", err)
	}
	if added {
		t.Fatal("expected a no-op for an already-imported module")
	}
}
