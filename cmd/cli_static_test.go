package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/static"
	"github.com/daqing/airway/lib/utils"
)

// staticStubComponent writes a minimal document containing the marker, and
// probes the pinned env through utils.AppConfig so tests can assert which
// asset URLs static:build renders.
func staticStubComponent(marker string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<html><title>"+marker+"</title>local="+fmt.Sprint(utils.AppConfig().IsLocal)+"</html>")
		return err
	})
}

func stubStaticPages(t *testing.T) {
	t.Helper()

	clearDSNEnv(t)

	staticPages = []static.Page{
		{Slug: "/", Component: staticStubComponent("home")},
		{Slug: "/about", Component: staticStubComponent("about")},
	}
	t.Cleanup(func() {
		staticPages = nil
		staticProviders = nil
	})
}

// clearDSNEnv keeps tests hermetic even on machines with a DSN exported:
// empty values make cliDSN fail, so setupExportDB skips database setup.
func clearDSNEnv(t *testing.T) {
	t.Helper()

	t.Setenv("AIRWAY_DSN", "")
	t.Setenv("DSN", "")
}

func seedDistBundle(t *testing.T) {
	t.Helper()

	if err := os.MkdirAll("app/assets/dist/css", 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	writeFile(t, "app/assets/dist/app.js", "bundle")
	writeFile(t, "app/assets/dist/css/app.css", "body{}")
}

func TestStaticBuildRequiresRegisteredPages(t *testing.T) {
	useTempWorkingDir(t)

	err := run([]string{"static:build"})
	if err == nil || !strings.Contains(err.Error(), "no static pages") {
		t.Fatalf("static:build without pages = %v, want a guidance error", err)
	}
}

func TestStaticBuildExportsPagesAndBundle(t *testing.T) {
	useTempWorkingDir(t)
	stubStaticPages(t)
	seedDistBundle(t)

	// The export must pin production asset URLs even when the developer
	// builds in local mode.
	t.Setenv("AIRWAY_ENV", "local")

	if err := run([]string{"static:build", "--out", "out"}); err != nil {
		t.Fatalf("static:build: %v", err)
	}

	assertContains(t, readFile(t, "out/index.html"), "<title>home</title>")
	assertContains(t, readFile(t, "out/index.html"), "local=false")
	assertContains(t, readFile(t, "out/about/index.html"), "<title>about</title>")
	assertContains(t, readFile(t, "out/assets/app.js"), "bundle")
	assertContains(t, readFile(t, "out/assets/css/app.css"), "body{}")
}

func TestStaticBuildRequiresFrontendBundle(t *testing.T) {
	useTempWorkingDir(t)
	stubStaticPages(t)

	err := run([]string{"static:build"})
	if err == nil || !strings.Contains(err.Error(), "js:build") {
		t.Fatalf("static:build without a bundle = %v, want a js:build hint", err)
	}
}

func TestStaticBuildArgValidation(t *testing.T) {
	useTempWorkingDir(t)
	stubStaticPages(t)
	seedDistBundle(t)

	if err := run([]string{"static:build", "--out=eq"}); err != nil {
		t.Fatalf("static:build --out=eq: %v", err)
	}
	if _, err := os.Stat("eq/index.html"); err != nil {
		t.Fatalf("expected eq/index.html: %v", err)
	}

	if err := run([]string{"static:build", "--bogus"}); err == nil {
		t.Fatalf("expected unknown argument to fail")
	}

	if err := run([]string{"static:build", "--out="}); err == nil {
		t.Fatalf("expected empty --out to fail")
	}

	if err := run([]string{"static:build", "-h"}); err != nil {
		t.Fatalf("static:build -h: %v", err)
	}
}

func TestStaticServeWithoutPagesFails(t *testing.T) {
	useTempWorkingDir(t)

	err := run([]string{"static:serve"})
	if err == nil || !strings.Contains(err.Error(), "no static pages") {
		t.Fatalf("static:serve without pages = %v, want a guidance error", err)
	}
}

func TestStaticServeArgValidation(t *testing.T) {
	useTempWorkingDir(t)
	stubStaticPages(t)

	// An unknown argument must fail before the server binds.
	if err := run([]string{"static:serve", "--bogus"}); err == nil {
		t.Fatalf("expected unknown static:serve argument to fail")
	}

	if err := run([]string{"static:serve", "-h"}); err != nil {
		t.Fatalf("static:serve -h: %v", err)
	}
}

func TestSetStaticPagesRejectsDuplicates(t *testing.T) {
	// The first page is appended before the duplicate panics.
	t.Cleanup(func() { staticPages = nil })

	mustPanic(t, "duplicate static page slug", func() {
		SetStaticPages(
			static.Page{Slug: "/about", Component: staticStubComponent("a")},
			static.Page{Slug: "about", Component: staticStubComponent("b")},
		)
	})
}

// staticExportRow mirrors a scaffolded model closely enough for repo
// queries: db tags plus a TableName method.
type staticExportRow struct {
	ID    int64  `db:"id"`
	Title string `db:"title"`
}

func (staticExportRow) TableName() string { return "static_export_rows" }

func staticRowComponent(title string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<html>"+title+"</html>")
		return err
	})
}

func TestStaticBuildCollectsProviderPages(t *testing.T) {
	useTempWorkingDir(t)
	stubStaticPages(t)
	seedDistBundle(t)

	SetStaticPagesProvider(func() ([]static.Page, error) {
		return []static.Page{
			{Slug: "/rows/1", Component: staticStubComponent("row-one")},
			{Slug: "/rows/2", Component: staticStubComponent("row-two")},
		}, nil
	})

	if err := run([]string{"static:build", "--out", "out"}); err != nil {
		t.Fatalf("static:build: %v", err)
	}

	assertContains(t, readFile(t, "out/rows/1/index.html"), "row-one")
	assertContains(t, readFile(t, "out/rows/2/index.html"), "row-two")
}

func TestStaticBuildProviderErrorFails(t *testing.T) {
	useTempWorkingDir(t)
	stubStaticPages(t)
	seedDistBundle(t)

	SetStaticPagesProvider(func() ([]static.Page, error) {
		return nil, fmt.Errorf("enumerate failed")
	})

	err := run([]string{"static:build"})
	if err == nil || !strings.Contains(err.Error(), "enumerate failed") {
		t.Fatalf("static:build with failing provider = %v, want the provider error", err)
	}
}

func TestStaticBuildProviderPanicBecomesError(t *testing.T) {
	useTempWorkingDir(t)
	stubStaticPages(t)
	seedDistBundle(t)

	// A provider that panics — for example repo queries hit before the
	// database is set up — must surface as an error, not a crash.
	SetStaticPagesProvider(func() ([]static.Page, error) {
		panic("database is not setup yet")
	})

	err := run([]string{"static:build"})
	if err == nil || !strings.Contains(err.Error(), "provider failed") {
		t.Fatalf("static:build with panicking provider = %v, want a wrapped error", err)
	}
}

func TestStaticBuildEnumeratesRowsFromDatabase(t *testing.T) {
	useTempWorkingDir(t)
	stubStaticPages(t)
	seedDistBundle(t)

	t.Setenv("DSN", "sqlite://export-test.db")

	db, err := repo.SetupDB("sqlite://export-test.db")
	if err != nil {
		t.Fatalf("SetupDB: %v", err)
	}

	if _, err := db.Conn().Exec(`CREATE TABLE static_export_rows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	for _, title := range []string{"hello", "world"} {
		if _, err := db.Conn().Exec(`INSERT INTO static_export_rows (title) VALUES (?)`, title); err != nil {
			t.Fatalf("insert row: %v", err)
		}
	}

	SetStaticPagesProvider(func() ([]static.Page, error) {
		items, err := repo.FindAll[staticExportRow]()
		if err != nil {
			return nil, err
		}

		pages := make([]static.Page, 0, len(items))
		for _, item := range items {
			row := item
			pages = append(pages, static.Page{
				Slug:      fmt.Sprintf("/rows/%d", row.ID),
				Component: staticRowComponent(row.Title),
			})
		}
		return pages, nil
	})

	if err := run([]string{"static:build", "--out", "out"}); err != nil {
		t.Fatalf("static:build: %v", err)
	}

	assertContains(t, readFile(t, "out/rows/1/index.html"), "hello")
	assertContains(t, readFile(t, "out/rows/2/index.html"), "world")
}

func mustPanic(t *testing.T, name string, fn func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatalf("%s: expected panic", name)
		}
	}()

	fn()
}
