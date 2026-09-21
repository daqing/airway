package migrate

import (
	"bytes"
	"database/sql"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	_ "modernc.org/sqlite"
)

func sqliteTestOptions(t *testing.T, migrations fs.FS, out io.Writer) Options {
	t.Helper()

	dsn := "sqlite://" + filepath.ToSlash(filepath.Join(t.TempDir(), "test.db"))
	return Options{DSN: dsn, Migrations: migrations, SnapshotPath: "", Out: out}
}

func testMigrations() fstest.MapFS {
	return fstest.MapFS{
		"0001_create_posts.up.sql":   {Data: []byte("CREATE TABLE posts (id INTEGER PRIMARY KEY, title VARCHAR(255) NOT NULL);")},
		"0001_create_posts.down.sql": {Data: []byte("DROP TABLE posts;")},
		"0002_add_body.up.sql":       {Data: []byte("ALTER TABLE posts ADD COLUMN body TEXT;")},
		"0002_add_body.down.sql":     {Data: []byte("ALTER TABLE posts DROP COLUMN body;")},
	}
}

func columnExists(t *testing.T, dsn string, table string, column string) bool {
	t.Helper()

	db, err := sql.Open("sqlite", dsn[len("sqlite://"):])
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM pragma_table_info(?) WHERE name = ?", table, column)
	if err != nil {
		t.Fatalf("pragma_table_info: %v", err)
	}
	defer rows.Close()

	return rows.Next()
}

func TestRunAppliesSQLMigrations(t *testing.T) {
	opts := sqliteTestOptions(t, testMigrations(), io.Discard)

	if err := Run(opts); err != nil {
		t.Fatalf("run: %v", err)
	}

	if columnExists(t, opts.DSN, "posts", "id") != true {
		t.Fatalf("expected posts table to exist after migrate")
	}
	if columnExists(t, opts.DSN, "posts", "body") != true {
		t.Fatalf("expected body column after second migration")
	}

	// Re-running is a no-op: applied versions are skipped.
	if err := Run(opts); err != nil {
		t.Fatalf("re-run: %v", err)
	}
}

func TestRunReportsAppliedUnitsOnReRun(t *testing.T) {
	var out bytes.Buffer
	opts := sqliteTestOptions(t, testMigrations(), &out)

	if err := Run(opts); err != nil {
		t.Fatalf("run: %v", err)
	}

	out.Reset()
	if err := Run(opts); err != nil {
		t.Fatalf("re-run: %v", err)
	}

	if !strings.Contains(out.String(), "0001_create_posts.up.sql already applied") {
		t.Fatalf("expected applied versions to be skipped, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "Already at the latest migration") {
		t.Fatalf("expected latest notice, got: %s", out.String())
	}
}

func TestRollbackRemovesLastMigration(t *testing.T) {
	opts := sqliteTestOptions(t, testMigrations(), io.Discard)

	if err := Run(opts); err != nil {
		t.Fatalf("run: %v", err)
	}

	if err := Rollback(opts, 1); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if columnExists(t, opts.DSN, "posts", "body") {
		t.Fatalf("expected body column dropped after rollback")
	}
	if !columnExists(t, opts.DSN, "posts", "id") {
		t.Fatalf("expected posts table to survive the rollback")
	}
}

func TestStatusListsPendingAndApplied(t *testing.T) {
	var out bytes.Buffer
	opts := sqliteTestOptions(t, testMigrations(), &out)

	if err := Status(opts); err != nil {
		t.Fatalf("status: %v", err)
	}
	if strings.Count(out.String(), "pending") != 2 {
		t.Fatalf("expected two pending units before migrate, got: %s", out.String())
	}

	if err := Run(opts); err != nil {
		t.Fatalf("run: %v", err)
	}

	out.Reset()
	if err := Status(opts); err != nil {
		t.Fatalf("status: %v", err)
	}
	if strings.Count(out.String(), "applied") != 2 || strings.Contains(out.String(), "pending") {
		t.Fatalf("expected all units applied after migrate, got: %s", out.String())
	}
}

func TestMissingDownMigrationIsRejected(t *testing.T) {
	fsys := fstest.MapFS{
		"0001_create_posts.up.sql": {Data: []byte("CREATE TABLE posts (id INTEGER PRIMARY KEY);")},
	}
	opts := sqliteTestOptions(t, fsys, io.Discard)

	err := Run(opts)
	if err == nil || !strings.Contains(err.Error(), "missing down migration") {
		t.Fatalf("expected missing down migration error, got: %v", err)
	}
}

func TestManagerUsesInjectedClockForTempTables(t *testing.T) {
	opts := sqliteTestOptions(t, testMigrations(), io.Discard)
	opts.Now = func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }

	if err := Run(opts); err != nil {
		t.Fatalf("run: %v", err)
	}

	if err := Rollback(opts, 1); err != nil {
		t.Fatalf("rollback with injected clock: %v", err)
	}
}

func TestOptionsValidation(t *testing.T) {
	if err := Run(Options{}); err == nil || !strings.Contains(err.Error(), "Migrations must be set") {
		t.Fatalf("expected nil-fs error, got: %v", err)
	}

	err := Run(Options{Migrations: testMigrations()})
	if err == nil || !strings.Contains(err.Error(), "DSN must be set") {
		t.Fatalf("expected empty-DSN error, got: %v", err)
	}
}
