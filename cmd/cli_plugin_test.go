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

func TestInstallPluginMigrationsFromDir(t *testing.T) {
	moduleDir := t.TempDir()
	migrateDir := filepath.Join(moduleDir, "host", "db", "migrate")
	makeDirs(t, migrateDir)
	writeFile(t, filepath.Join(migrateDir, "20240101000000_create_messages.up.sql"), "CREATE TABLE messages (id INTEGER PRIMARY KEY);")
	writeFile(t, filepath.Join(migrateDir, "20240101000000_create_messages.down.sql"), "DROP TABLE messages;")

	dstDir := filepath.Join(t.TempDir(), "db", "migrate")

	if err := installPluginMigrationsFromDir("im", moduleDir, dstDir); err != nil {
		t.Fatalf("install plugin migrations from dir: %v", err)
	}

	up := migrationFiles(t, dstDir, "*_create_messages.up.sql")
	down := migrationFiles(t, dstDir, "*_create_messages.down.sql")
	if len(up) != 1 || len(down) != 1 {
		t.Fatalf("expected one up/down pair, got %d up and %d down files", len(up), len(down))
	}
	if got := readFile(t, up[0]); got != "CREATE TABLE messages (id INTEGER PRIMARY KEY);" {
		t.Fatalf("unexpected up migration content: %s", got)
	}

	// Installing again must skip instead of duplicating.
	if err := installPluginMigrationsFromDir("im", moduleDir, dstDir); err != nil {
		t.Fatalf("reinstall plugin migrations: %v", err)
	}

	if got := len(migrationFiles(t, dstDir, "*.sql")); got != 2 {
		t.Fatalf("expected 2 migration files after reinstall, got %d", got)
	}
}

// The plugin module root's db/migrate is no longer read; migrations must live
// under host/db/migrate.
func TestInstallPluginMigrationsFromDirIgnoresLegacyLocation(t *testing.T) {
	moduleDir := t.TempDir()
	legacyDir := filepath.Join(moduleDir, "db", "migrate")
	makeDirs(t, legacyDir)
	writeFile(t, filepath.Join(legacyDir, "20240101000000_create_messages.up.sql"), "CREATE TABLE messages (id INTEGER PRIMARY KEY);")
	writeFile(t, filepath.Join(legacyDir, "20240101000000_create_messages.down.sql"), "DROP TABLE messages;")

	dstDir := filepath.Join(t.TempDir(), "db", "migrate")

	if err := installPluginMigrationsFromDir("im", moduleDir, dstDir); err != nil {
		t.Fatalf("install plugin migrations from dir: %v", err)
	}

	if got := len(migrationFiles(t, dstDir, "*.sql")); got != 0 {
		t.Fatalf("expected no migrations from the legacy location, got %d files", got)
	}
}

func TestInstallPluginMigrationsFromDirWithoutDirIsNoOp(t *testing.T) {
	dstDir := filepath.Join(t.TempDir(), "db", "migrate")

	if err := installPluginMigrationsFromDir("im", t.TempDir(), dstDir); err != nil {
		t.Fatalf("expected missing host/db/migrate to be a no-op, got: %v", err)
	}
}

func migrationFiles(t *testing.T, dir, pattern string) []string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestHasGoCodeMigrations(t *testing.T) {
	cases := []struct {
		name  string
		files fstest.MapFS
		want  bool
	}{
		{"go files", fstest.MapFS{"users.go": {Data: []byte("package migrate")}}, true},
		{"test files only", fstest.MapFS{"migrate_test.go": {Data: []byte("package migrate")}}, false},
		{"sql files only", fstest.MapFS{
			"20240101000000_create_messages.up.sql":   {Data: []byte("SELECT 1;")},
			"20240101000000_create_messages.down.sql": {Data: []byte("SELECT 1;")},
		}, false},
		{"sql and go mixed", fstest.MapFS{
			"20240101000000_create_messages.up.sql": {Data: []byte("SELECT 1;")},
			"users.go":                              {Data: []byte("package migrate")},
		}, true},
	}

	for _, tt := range cases {
		if got := hasGoCodeMigrations(tt.files); got != tt.want {
			t.Fatalf("%s: hasGoCodeMigrations = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// A plugin whose migrations are all written in Go code (no SQL pairs)
// installs nothing and reports that they take effect on import instead of
// claiming none exist.
func TestInstallPluginMigrationsWithOnlyGoCode(t *testing.T) {
	migrations := fstest.MapFS{
		"users.go":        {Data: []byte("package migrate")},
		"migrate_test.go": {Data: []byte("package migrate")},
	}

	dstDir := filepath.Join(t.TempDir(), "db", "migrate")

	if err := installPluginMigrations("im", migrations, dstDir, time.Now()); err != nil {
		t.Fatalf("install plugin migrations: %v", err)
	}

	if _, err := os.Stat(dstDir); !os.IsNotExist(err) {
		t.Fatalf("expected no destination directory for a plugin with only Go-code migrations, got: %v", err)
	}
}

func TestEnsurePluginImportAppendsImportBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugins.go")
	// The scaffolded plugins.go shows a sample import block inside a comment;
	// the edit must not match it.
	writeFile(t, path, `package main

// Plugins are optional feature modules shipped as independent Go modules
// (see docs/plugin.md). Enable one by adding a blank import below and
// running `+"`go mod tidy`"+`:
//
//	import (
//		_ "github.com/example/airway-im-plugin"
//	)
//
// The plugin's package init registers it with lib/plugin.
`)

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

func TestModulePathAt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/daqing/airway-im-plugin\n\ngo 1.26\n")

	module, err := modulePathAt(dir)
	if err != nil {
		t.Fatalf("module path: %v", err)
	}
	if module != "github.com/daqing/airway-im-plugin" {
		t.Fatalf("modulePathAt = %q", module)
	}

	if _, err := modulePathAt(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("expected an error for a directory without go.mod")
	}
}

func TestInstallPluginDeps(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "deps")
	makeDirs(t, filepath.Join(srcDir, "im", "app"))
	makeDirs(t, filepath.Join(srcDir, "im", "gateway"))
	writeFile(t, filepath.Join(srcDir, ".keep"), "")
	writeFile(t, filepath.Join(srcDir, "im", "app", "docker-compose.yml"), "services: {}\n")
	writeFile(t, filepath.Join(srcDir, "im", "gateway", "main.go"), "package main\n")
	writeFile(t, filepath.Join(srcDir, "im", "gateway", "go.mod.templ"), "module gateway\n\ngo 1.26\n")

	// The scaffolded host project ships an empty deps/ directory.
	dstRoot := filepath.Join(t.TempDir(), "deps")
	makeDirs(t, dstRoot)
	writeFile(t, filepath.Join(dstRoot, ".keep"), "host\n")

	if err := installPluginDepsFrom("im", srcDir, dstRoot); err != nil {
		t.Fatalf("install plugin deps: %v", err)
	}

	if got := readFile(t, filepath.Join(dstRoot, "im", "app", "docker-compose.yml")); got != "services: {}\n" {
		t.Fatalf("unexpected im/app/docker-compose.yml content: %s", got)
	}
	if got := readFile(t, filepath.Join(dstRoot, "im", "gateway", "main.go")); got != "package main\n" {
		t.Fatalf("unexpected im/gateway/main.go content: %s", got)
	}
	// The .templ suffix is stripped on install.
	if got := readFile(t, filepath.Join(dstRoot, "im", "gateway", "go.mod")); got != "module gateway\n\ngo 1.26\n" {
		t.Fatalf("unexpected im/gateway/go.mod content: %s", got)
	}
	if _, err := os.Stat(filepath.Join(dstRoot, "im", "gateway", "go.mod.templ")); !os.IsNotExist(err) {
		t.Fatal("expected no go.mod.templ in the destination")
	}
	// The plugin's own deps/.keep is skipped; the host's stays untouched.
	if got := readFile(t, filepath.Join(dstRoot, ".keep")); got != "host\n" {
		t.Fatalf("expected the host's deps/.keep untouched, got: %s", got)
	}

	// Reinstalling skips existing files instead of overwriting local edits.
	writeFile(t, filepath.Join(dstRoot, "im", "app", "docker-compose.yml"), "services: edited\n")
	if err := installPluginDepsFrom("im", srcDir, dstRoot); err != nil {
		t.Fatalf("reinstall plugin deps: %v", err)
	}
	if got := readFile(t, filepath.Join(dstRoot, "im", "app", "docker-compose.yml")); got != "services: edited\n" {
		t.Fatalf("expected existing file untouched, got: %s", got)
	}
}

// A bare file beside its .templ variant (go.mod next to go.mod.templ, so a
// nested module builds in the plugin checkout) is skipped: the .templ variant
// is the distribution copy, matching what a module-zip download delivers.
func TestInstallPluginDepsPrefersTemplVariant(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "deps")
	makeDirs(t, filepath.Join(srcDir, "im", "delivery"))
	writeFile(t, filepath.Join(srcDir, "im", "delivery", "go.mod"), "module delivery\n")
	writeFile(t, filepath.Join(srcDir, "im", "delivery", "go.mod.templ"), "module delivery\n")

	dstRoot := filepath.Join(t.TempDir(), "deps")

	if err := installPluginDepsFrom("im", srcDir, dstRoot); err != nil {
		t.Fatalf("install plugin deps: %v", err)
	}

	if got := readFile(t, filepath.Join(dstRoot, "im", "delivery", "go.mod")); got != "module delivery\n" {
		t.Fatalf("unexpected im/delivery/go.mod content: %s", got)
	}
}

// A bare go.mod that drifted from its go.mod.templ fails the install: the
// nested module would build against something else than what ships.
func TestInstallPluginDepsRejectsDriftedGoMod(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "deps")
	makeDirs(t, filepath.Join(srcDir, "im", "delivery"))
	writeFile(t, filepath.Join(srcDir, "im", "delivery", "go.mod"), "module delivery-stale\n")
	writeFile(t, filepath.Join(srcDir, "im", "delivery", "go.mod.templ"), "module delivery-dist\n")

	err := installPluginDepsFrom("im", srcDir, filepath.Join(t.TempDir(), "deps"))
	if err == nil {
		t.Fatal("expected an error for a bare go.mod that drifted from its .templ variant")
	}
	if !strings.Contains(err.Error(), "im/delivery/go.mod") {
		t.Fatalf("expected the drifted file in the error, got: %v", err)
	}
}

// Reinstalling a bare go.mod kept in sync with its go.mod.templ skips the
// destination silently: nothing changed since the previous install. Files
// without such a twin still report the skip.
func TestInstallPluginDepsSkipsInSyncGoModSilently(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "deps")
	makeDirs(t, filepath.Join(srcDir, "im", "gateway"))
	makeDirs(t, filepath.Join(srcDir, "im", "app"))
	writeFile(t, filepath.Join(srcDir, "im", "gateway", "go.mod"), "module gateway\n")
	writeFile(t, filepath.Join(srcDir, "im", "gateway", "go.mod.templ"), "module gateway\n")
	writeFile(t, filepath.Join(srcDir, "im", "app", "docker-compose.yml"), "services: {}\n")

	dstRoot := filepath.Join(t.TempDir(), "deps")

	if err := installPluginDepsFrom("im", srcDir, dstRoot); err != nil {
		t.Fatalf("install plugin deps: %v", err)
	}

	output := captureStdout(t, func() {
		if err := installPluginDepsFrom("im", srcDir, dstRoot); err != nil {
			t.Fatalf("reinstall plugin deps: %v", err)
		}
	})

	if strings.Contains(output, "im/gateway/go.mod already exists") {
		t.Fatalf("expected no skip notice for the in-sync go.mod, got: %s", output)
	}
	if !strings.Contains(output, "im/app/docker-compose.yml already exists") {
		t.Fatalf("expected a skip notice for the unrelated existing file, got: %s", output)
	}
}

func TestInstallPluginDepsCreatesMissingHostDir(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "deps")
	makeDirs(t, filepath.Join(srcDir, "im", "app"))
	writeFile(t, filepath.Join(srcDir, "im", "app", "docker-compose.yml"), "services: {}\n")

	// Hosts scaffolded by older Airway versions have no deps/ directory.
	dstRoot := filepath.Join(t.TempDir(), "deps")

	if err := installPluginDepsFrom("im", srcDir, dstRoot); err != nil {
		t.Fatalf("install plugin deps: %v", err)
	}

	if got := readFile(t, filepath.Join(dstRoot, "im", "app", "docker-compose.yml")); got != "services: {}\n" {
		t.Fatalf("unexpected im/app/docker-compose.yml content: %s", got)
	}
}

func TestInstallPluginDepsCreatesHostDirForKeepOnlyDeps(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "deps")
	makeDirs(t, srcDir)
	writeFile(t, filepath.Join(srcDir, ".keep"), "")

	// Even a .keep-only plugin deps/ must leave the host with a deps/ directory.
	dstRoot := filepath.Join(t.TempDir(), "deps")

	if err := installPluginDepsFrom("im", srcDir, dstRoot); err != nil {
		t.Fatalf("install plugin deps: %v", err)
	}

	if info, err := os.Stat(dstRoot); err != nil || !info.IsDir() {
		t.Fatalf("expected the host deps/ directory to be created: %v", err)
	}

	entries, err := os.ReadDir(dstRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected the host deps/ directory to stay empty, got %d entries", len(entries))
	}
}

func TestInstallPluginDepsWithoutDirIsNoOp(t *testing.T) {
	if err := installPluginDepsFrom("im", filepath.Join(t.TempDir(), "deps"), t.TempDir()); err != nil {
		t.Fatalf("expected missing deps dir to be a no-op, got: %v", err)
	}
}

func TestInstallPluginHost(t *testing.T) {
	srcDir := filepath.Join(t.TempDir(), "host")
	makeDirs(t, filepath.Join(srcDir, "db", "migrate"))
	writeFile(t, filepath.Join(srcDir, "db", "migrate", "create_users.up.sql"), "CREATE TABLE users (id INTEGER);\n")
	writeFile(t, filepath.Join(srcDir, "docker-compose.yml"), "services: {}\n")
	writeFile(t, filepath.Join(srcDir, "Dockerfile.templ"), "FROM scratch\n")

	dstRoot := t.TempDir()

	if err := installPluginHostFrom("im", srcDir, dstRoot); err != nil {
		t.Fatalf("install plugin host tree: %v", err)
	}

	if got := readFile(t, filepath.Join(dstRoot, "docker-compose.yml")); got != "services: {}\n" {
		t.Fatalf("unexpected docker-compose.yml content: %s", got)
	}
	// The .templ suffix is stripped on install.
	if got := readFile(t, filepath.Join(dstRoot, "Dockerfile")); got != "FROM scratch\n" {
		t.Fatalf("unexpected Dockerfile content: %s", got)
	}
	// db/migrate is the migration installer's territory, not mirrored verbatim.
	if _, err := os.Stat(filepath.Join(dstRoot, "db")); !os.IsNotExist(err) {
		t.Fatal("expected no db/ tree in the destination")
	}

	// Reinstalling skips existing files instead of overwriting local edits.
	writeFile(t, filepath.Join(dstRoot, "docker-compose.yml"), "services: edited\n")
	if err := installPluginHostFrom("im", srcDir, dstRoot); err != nil {
		t.Fatalf("reinstall plugin host tree: %v", err)
	}
	if got := readFile(t, filepath.Join(dstRoot, "docker-compose.yml")); got != "services: edited\n" {
		t.Fatalf("expected existing file untouched, got: %s", got)
	}
}

func TestInstallPluginHostWithoutDirIsNoOp(t *testing.T) {
	if err := installPluginHostFrom("im", filepath.Join(t.TempDir(), "host"), t.TempDir()); err != nil {
		t.Fatalf("expected missing host dir to be a no-op, got: %v", err)
	}
}
