package cmd

import (
	"context"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/a-h/templ"

	"github.com/daqing/airway/lib/ssg"
)

// ssgStubTheme writes the page title into a minimal document and ships one
// stylesheet, enough to exercise ssg:build end to end.
type ssgStubTheme struct{}

func (ssgStubTheme) Name() string { return "stub" }

func (ssgStubTheme) Render(ctx *ssg.Context) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<html><title>"+ctx.Page.Title+"</title></html>")
		return err
	})
}

func (ssgStubTheme) Assets() fs.FS {
	return fstest.MapFS{"css/site.css": &fstest.MapFile{Data: []byte("body{}")}}
}

func stubSSGBuilder(t *testing.T) {
	t.Helper()

	ssgBuilder = func() (*ssg.Site, error) {
		return ssg.New(ssg.Meta{Title: "Demo"}).
			Use(ssgStubTheme{}).
			Page("/", "Home", nil).
			Page("/about", "About", nil), nil
	}
	t.Cleanup(func() { ssgBuilder = nil })
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Fatalf("%q missing %q", got, want)
	}
}

func TestSiteBuildRequiresRegisteredBuilder(t *testing.T) {
	useTempWorkingDir(t)

	err := run([]string{"ssg:build"})
	if err == nil || !strings.Contains(err.Error(), "no site is defined") {
		t.Fatalf("ssg:build without a builder = %v, want a guidance error", err)
	}
}

func TestSiteBuildExportsPagesAndAssets(t *testing.T) {
	useTempWorkingDir(t)
	stubSSGBuilder(t)

	if err := run([]string{"ssg:build", "--out", "out"}); err != nil {
		t.Fatalf("ssg:build: %v", err)
	}

	assertContains(t, readFile(t, "out/index.html"), "<title>Home</title>")
	assertContains(t, readFile(t, "out/about/index.html"), "<title>About</title>")
	assertContains(t, readFile(t, "out/assets/css/site.css"), "body{}")
}

func TestSiteBuildOutFormsAndErrors(t *testing.T) {
	useTempWorkingDir(t)
	stubSSGBuilder(t)

	if err := run([]string{"ssg:build", "--out=eq"}); err != nil {
		t.Fatalf("ssg:build --out=eq: %v", err)
	}
	if _, err := os.Stat("eq/index.html"); err != nil {
		t.Fatalf("expected eq/index.html: %v", err)
	}

	if err := run([]string{"ssg:build", "--bogus"}); err == nil {
		t.Fatalf("expected unknown argument to fail")
	}

	if err := run([]string{"ssg:build", "--out="}); err == nil {
		t.Fatalf("expected empty --out to fail")
	}

	if err := run([]string{"ssg:build", "-h"}); err != nil {
		t.Fatalf("ssg:build -h: %v", err)
	}
}

func TestSiteServeArgValidation(t *testing.T) {
	useTempWorkingDir(t)
	stubSSGBuilder(t)

	// An unknown argument must fail before the server binds.
	if err := run([]string{"ssg:serve", "--bogus"}); err == nil {
		t.Fatalf("expected unknown ssg:serve argument to fail")
	}

	if err := run([]string{"ssg:serve", "-h"}); err != nil {
		t.Fatalf("ssg:serve -h: %v", err)
	}
}

func TestSiteServeWithoutBuilderFails(t *testing.T) {
	useTempWorkingDir(t)

	err := run([]string{"ssg:serve"})
	if err == nil || !strings.Contains(err.Error(), "no site is defined") {
		t.Fatalf("ssg:serve without a builder = %v, want a guidance error", err)
	}
}
