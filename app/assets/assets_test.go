package assets

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daqing/airway/lib/jsbuild"
)

func TestEmbeddedBundleIsServed(t *testing.T) {
	handler := Handler()

	get := func(path, ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	rec := get("/assets/"+jsbuild.EntryJS, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("app.js status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Errorf("content type %q", ct)
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("bare path missing ETag")
	}
	if rec := get("/assets/"+jsbuild.EntryJS, etag); rec.Code != http.StatusNotModified {
		t.Errorf("revalidation status %d, want 304", rec.Code)
	}

	rec = get("/assets/"+jsbuild.EntryJS+"?v=anything", "")
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("versioned path cache-control %q, want immutable", cc)
	}

	rec = get("/assets/"+jsbuild.ManifestFile, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "json") {
		t.Errorf("manifest status %d type %q", rec.Code, rec.Header().Get("Content-Type"))
	}

	if rec := get("/assets/nope.js", ""); rec.Code != http.StatusNotFound {
		t.Errorf("missing asset status %d, want 404", rec.Code)
	}
	if rec := get("/assets/..%2fgo.mod", ""); rec.Code != http.StatusNotFound {
		t.Errorf("traversal attempt status %d, want 404", rec.Code)
	}
}

func TestManifestAndEntryPath(t *testing.T) {
	m, err := Manifest()
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if m.Entry != jsbuild.EntryJS || m.Hash == "" {
		t.Errorf("embedded manifest incomplete: %+v", m)
	}

	p := EntryPath()
	if !strings.HasPrefix(p, "/assets/"+jsbuild.EntryJS+"?v=") {
		t.Errorf("EntryPath %q lacks cache buster", p)
	}
}

func TestStaticHandlerServesFilesFromRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "download.zip"), []byte("extra payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}

	handler := StaticHandler(dir)

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	if rec := get("/static/download.zip"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "extra payload") {
		t.Errorf("static file: status %d body %q", rec.Code, rec.Body.String())
	}
	if rec := get("/static/missing.zip"); rec.Code != http.StatusNotFound {
		t.Errorf("missing file status %d, want 404", rec.Code)
	}
	if rec := get("/static/nested"); rec.Code != http.StatusNotFound {
		t.Errorf("directory status %d, want 404", rec.Code)
	}
	if rec := get("/static/../go.mod"); rec.Code != http.StatusNotFound {
		t.Errorf("traversal attempt status %d, want 404", rec.Code)
	}
}

func TestStaticHandlerDisabledWithoutRoot(t *testing.T) {
	rec := httptest.NewRecorder()
	StaticHandler("").ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/app.js", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status %d, want 404", rec.Code)
	}
}

func TestAssetsHandlerIgnoresStaticPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "download.zip"), []byte("extra payload"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STATIC_PATH", dir)

	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/download.zip", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("/assets must serve only the embedded bundle: status %d, want 404", rec.Code)
	}
}
