package jsbuild

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	entry := filepath.Join(root, SourceDir)
	if err := os.MkdirAll(entry, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entry, "app.tsx"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestBuildWritesDistAndManifest(t *testing.T) {
	root := writeFixture(t, `console.log("phase2")`)

	m1, err := Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if m1.Entry != EntryJS {
		t.Errorf("entry = %q, want %q", m1.Entry, EntryJS)
	}
	if len(m1.Hash) != 16 {
		t.Errorf("hash %q is not 16 hex chars", m1.Hash)
	}
	found := map[string]bool{}
	for _, f := range m1.Files {
		found[f] = true
	}
	if !found[EntryJS] || !found[EntryJS+".map"] {
		t.Errorf("files missing bundle or sourcemap: %v", m1.Files)
	}

	bundle, err := os.ReadFile(filepath.Join(root, DistDir, EntryJS))
	if err != nil || !strings.Contains(string(bundle), "phase2") {
		t.Fatalf("dist bundle wrong: %q %v", bundle, err)
	}
	var onDisk Manifest
	data, err := os.ReadFile(filepath.Join(root, DistDir, ManifestFile))
	if err != nil {
		t.Fatalf("manifest not written: %v", err)
	}
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatalf("manifest invalid: %v", err)
	}
	if onDisk.Hash != m1.Hash {
		t.Errorf("manifest hash %q != returned %q", onDisk.Hash, m1.Hash)
	}

	// same input, same hash: committed dist diffs stay minimal
	m2, err := Build(root)
	if err != nil {
		t.Fatalf("second Build: %v", err)
	}
	if m2.Hash != m1.Hash {
		t.Errorf("build not deterministic: %q vs %q", m1.Hash, m2.Hash)
	}
}

func TestDevServerServesRevalidatesAndRebuilds(t *testing.T) {
	root := writeFixture(t, `console.log("v1")`)

	broadcasts := make(chan string, 4)
	dev, err := StartDev(root, func(msg string) { broadcasts <- msg })
	if err != nil {
		t.Fatalf("StartDev: %v", err)
	}
	defer dev.Close()

	handler := dev.Handler()

	get := func(path, ifNoneMatch string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	rec := get("/assets/app.js", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("app.js status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Errorf("content type %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "v1") {
		t.Errorf("bundle does not contain v1 source")
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("missing ETag")
	}

	if rec := get("/assets/app.js", etag); rec.Code != http.StatusNotModified {
		t.Errorf("revalidation status %d, want 304", rec.Code)
	}

	rec = get("/assets/livereload.js", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "WebSocket") {
		t.Errorf("livereload.js not served: %d", rec.Code)
	}

	// change the source; the mtime poller should rebuild and broadcast
	if err := os.WriteFile(filepath.Join(root, SourceDir, "app.tsx"), []byte(`console.log("v2")`), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-broadcasts:
	case <-time.After(5 * time.Second):
		t.Fatal("no livereload broadcast after source change")
	}

	rec = get("/assets/app.js", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("app.js after rebuild: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "v2") {
		t.Errorf("bundle did not pick up new source: %q", rec.Body.String())
	}
	if rec.Header().Get("ETag") == etag {
		t.Errorf("ETag unchanged across rebuilds")
	}

	if rec := get("/assets/missing.js", ""); rec.Code != http.StatusNotFound {
		t.Errorf("missing asset status %d, want 404", rec.Code)
	}
}

func TestMissingVendorHint(t *testing.T) {
	jsx := `export default () => <div class="island-counter" />`
	writeManifest := func(root string) {
		if err := os.WriteFile(filepath.Join(root, "js.pkg.json"), []byte(`{"deps":{"preact":"10.29.8"}}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("build with declared deps", func(t *testing.T) {
		root := writeFixture(t, jsx)
		writeManifest(root)

		_, err := Build(root)
		if !errors.Is(err, ErrVendorMissing) {
			t.Fatalf("Build error = %v, want ErrVendorMissing", err)
		}
	})

	t.Run("dev server with declared deps", func(t *testing.T) {
		root := writeFixture(t, jsx)
		writeManifest(root)

		if _, err := StartDev(root, nil); !errors.Is(err, ErrVendorMissing) {
			t.Fatalf("StartDev error = %v, want ErrVendorMissing", err)
		}
	})

	t.Run("no declared deps falls through to esbuild", func(t *testing.T) {
		root := writeFixture(t, jsx)

		_, err := Build(root)
		if err == nil || !strings.Contains(err.Error(), "preact/compat") || strings.Contains(err.Error(), "js:install") {
			t.Fatalf("Build error = %v, want esbuild resolution error without hint", err)
		}
	})

	t.Run("vendor present passes the check", func(t *testing.T) {
		root := writeFixture(t, jsx)
		writeManifest(root)
		if err := os.MkdirAll(filepath.Join(root, VendorDir), 0o755); err != nil {
			t.Fatal(err)
		}

		_, err := Build(root)
		if err == nil || strings.Contains(err.Error(), "js:install") {
			t.Fatalf("Build error = %v, want esbuild error (empty vendor), not the hint", err)
		}
	})
}

func TestScanMtimesIgnoresVendor(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("app.tsx", "1")
	write("demos/a.tsx", "1")
	write("vendor/preact/package.json", "{}")
	write("notes.md", "not watched")

	got := scanMtimes(dir)
	if len(got) != 2 {
		t.Errorf("scanMtimes = %v, want only app.tsx and demos/a.tsx", got)
	}
}
