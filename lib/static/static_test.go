package static

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

// stubComponent writes a minimal document containing the marker, so tests
// can assert which component landed in which file.
func stubComponent(marker string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, "<html>"+marker+"</html>")
		return err
	})
}

func stubPages() []Page {
	return []Page{
		{Slug: "/", Component: stubComponent("home")},
		{Slug: "/about", Component: stubComponent("about")},
	}
}

func TestNormalizeSlugMatchesSSG(t *testing.T) {
	for _, tc := range []struct {
		slug string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"about", "/about"},
		{"/about/", "/about"},
	} {
		if got := NormalizeSlug(tc.slug); got != tc.want {
			t.Fatalf("NormalizeSlug(%q) = %q, want %q", tc.slug, got, tc.want)
		}
	}
}

func TestBuildWritesPages(t *testing.T) {
	out := t.TempDir()

	if err := Build(stubPages(), out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	assertContains(t, readFile(t, out+"/index.html"), "home")
	assertContains(t, readFile(t, out+"/about/index.html"), "about")
}

func TestBuildNormalizesSlugs(t *testing.T) {
	out := t.TempDir()

	pages := []Page{
		{Slug: "docs/guide", Component: stubComponent("guide")},
		{Slug: "about/", Component: stubComponent("about")},
	}

	if err := Build(pages, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	assertContains(t, readFile(t, out+"/docs/guide/index.html"), "guide")
	assertContains(t, readFile(t, out+"/about/index.html"), "about")
}

func TestBuildOverwritesPagesButKeepsOtherFiles(t *testing.T) {
	out := t.TempDir()
	keep := out + "/keep.txt"
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatalf("seed keep.txt: %v", err)
	}

	if err := Build(stubPages(), out); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := Build(stubPages(), out); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	assertContains(t, readFile(t, keep), "keep")
}

func TestBuildRejectsDuplicateSlugs(t *testing.T) {
	pages := append(stubPages(), Page{Slug: "", Component: stubComponent("again")})

	if err := Build(pages, t.TempDir()); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("Build with duplicate slug = %v, want a duplicate error", err)
	}
}

func TestBuildRejectsNilComponent(t *testing.T) {
	pages := []Page{{Slug: "/", Component: nil}}

	if err := Build(pages, t.TempDir()); err == nil || !strings.Contains(err.Error(), "no component") {
		t.Fatalf("Build with nil component = %v, want a component error", err)
	}
}

func TestBuildRejectsEmptyOutDir(t *testing.T) {
	if err := Build(stubPages(), ""); err == nil {
		t.Fatalf("expected Build with an empty outDir to fail")
	}
}

func TestBuildPropagatesRenderErrors(t *testing.T) {
	failing := templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
		return io.ErrClosedPipe
	})

	if err := Build([]Page{{Slug: "/", Component: failing}}, t.TempDir()); err == nil {
		t.Fatalf("expected Build to propagate the render error")
	}
}

func TestHandlerServesPages(t *testing.T) {
	server := httptest.NewServer(Handler(stubPages()))
	t.Cleanup(server.Close)

	tests := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/", http.StatusOK, "home"},
		{"/about", http.StatusOK, "about"},
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
