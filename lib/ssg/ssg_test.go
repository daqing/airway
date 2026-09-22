package ssg

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/a-h/templ"
)

// stubTheme renders the page title and slug into a full minimal document, so
// the tests can assert on both.
type stubTheme struct {
	assets fs.FS
}

func (t stubTheme) Name() string { return "stub" }

func (t stubTheme) Render(ctx *Context) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<html><title>"+ctx.Page.Title+"</title><nav>")
		for _, p := range ctx.Pages {
			_, err = io.WriteString(w, "<a href=\""+p.Slug+"\">"+p.Title+"</a>")
		}
		_, err = io.WriteString(w, "</nav><body>"+ctx.Page.Slug+"</body></html>")
		return err
	})
}

func (t stubTheme) Assets() fs.FS { return t.assets }

func stubSite(t *testing.T) *Site {
	t.Helper()

	theme := stubTheme{assets: fstest.MapFS{
		"css/site.css": &fstest.MapFile{Data: []byte("body{}")},
	}}

	return New(Meta{Title: "Stub", Language: "en"}).
		Use(theme).
		Page("/", "Home", nil).
		Page("/about", "About", nil)
}

func TestNormalizeSlug(t *testing.T) {
	for _, tc := range []struct {
		slug string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"about", "/about"},
		{"/about", "/about"},
		{"/about/", "/about"},
		{" /about/ ", "/about"},
		{"/a/b/", "/a/b"},
	} {
		if got := NormalizeSlug(tc.slug); got != tc.want {
			t.Fatalf("NormalizeSlug(%q) = %q, want %q", tc.slug, got, tc.want)
		}
	}
}

func TestSitePageValidation(t *testing.T) {
	mustPanic(t, "duplicate slug", func() {
		New(Meta{}).Page("/", "Home", nil).Page("/", "Again", nil)
	})
	mustPanic(t, "duplicate nested slug", func() {
		New(Meta{}).Page("/about", "About", nil).Page("about", "Again", nil)
	})
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

func TestBuildExportsPagesAndAssets(t *testing.T) {
	s := stubSite(t)
	out := t.TempDir()

	if err := Build(s, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	assertContains(t, readFile(t, out+"/index.html"), "<title>Home</title>")
	assertContains(t, readFile(t, out+"/about/index.html"), "<body>/about</body>")
	assertContains(t, readFile(t, out+"/assets/css/site.css"), "body{}")
}

func TestBuildWithoutThemeFails(t *testing.T) {
	if err := Build(New(Meta{}).Page("/", "Home", nil), t.TempDir()); err == nil {
		t.Fatalf("expected Build without a theme to fail")
	}
}

func TestHandlerServesPagesAndAssets(t *testing.T) {
	server := httptest.NewServer(Handler(stubSite(t)))
	t.Cleanup(server.Close)

	tests := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/", http.StatusOK, "<title>Home</title>"},
		{"/about", http.StatusOK, "<body>/about</body>"},
		{"/assets/css/site.css", http.StatusOK, "body{}"},
		{"/missing", http.StatusNotFound, ""},
	}

	client := server.Client()
	for _, tc := range tests {
		resp, err := client.Get(server.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != tc.wantStatus {
			t.Fatalf("GET %s status = %d, want %d", tc.path, resp.StatusCode, tc.wantStatus)
		}
		if tc.wantBody != "" && !strings.Contains(string(body), tc.wantBody) {
			t.Fatalf("GET %s body %q missing %q", tc.path, body, tc.wantBody)
		}
	}
}

func TestMetaLangDefaultsToEnglish(t *testing.T) {
	if got := (Meta{}).Lang(); got != "en" {
		t.Fatalf("Meta{}.Lang() = %q, want %q", got, "en")
	}
	if got := (Meta{Language: "zh-CN"}).Lang(); got != "zh-CN" {
		t.Fatalf("Meta{Language}.Lang() = %q, want %q", got, "zh-CN")
	}
}

func TestPagesReturnsCopy(t *testing.T) {
	s := stubSite(t)

	pages := s.Pages()
	pages[0].Title = "Mutated"

	if s.Pages()[0].Title != "Home" {
		t.Fatalf("Pages() must return a copy, got mutation: %q", s.Pages()[0].Title)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Fatalf("%q missing %q", got, want)
	}
}
