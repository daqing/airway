package assets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daqing/airway/lib/jsbuild"
)

func TestEmbeddedBundleIsServed(t *testing.T) {
	handler := Handler()

	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	rec := get("/assets/" + jsbuild.EntryJS)
	if rec.Code != http.StatusOK {
		t.Fatalf("app.js status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Errorf("content type %q", ct)
	}

	rec = get("/assets/" + jsbuild.EntryJS + "?v=anything")
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("versioned path cache-control %q, want immutable", cc)
	}

	if rec := get("/assets/nope.js"); rec.Code != http.StatusNotFound {
		t.Errorf("missing asset status %d, want 404", rec.Code)
	}
	if rec := get("/assets/..%2fgo.mod"); rec.Code != http.StatusNotFound {
		t.Errorf("traversal attempt status %d, want 404", rec.Code)
	}
}

func TestManifestAndEntryPath(t *testing.T) {
	m, err := Manifest()
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if m.Entry != jsbuild.EntryJS || len(m.Files) == 0 {
		t.Errorf("embedded manifest incomplete: %+v", m)
	}

	// The committed template bundle is a placeholder, so its manifest hash may
	// be empty; EntryPath must fall back to the un-busted path in that case.
	if p := EntryPath(); !strings.HasPrefix(p, "/assets/"+jsbuild.EntryJS) {
		t.Errorf("EntryPath %q", p)
	}
}
