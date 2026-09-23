package utils

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestCopyFSWritesNestedFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"app.js":              &fstest.MapFile{Data: []byte("bundle")},
		"css/airway.css":      &fstest.MapFile{Data: []byte("body{}")},
		"vendor/lib/index.js": &fstest.MapFile{Data: []byte("lib")},
	}

	dest := t.TempDir()
	if err := CopyFS(fsys, dest); err != nil {
		t.Fatalf("CopyFS: %v", err)
	}

	for path, want := range map[string]string{
		"app.js":                             "bundle",
		"css/airway.css":                     "body{}",
		filepath.Join("vendor/lib/index.js"): "lib",
	} {
		data, err := os.ReadFile(filepath.Join(dest, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(data) != want {
			t.Fatalf("%s = %q, want %q", path, data, want)
		}
	}
}

func TestCopyFSOverwritesExistingFiles(t *testing.T) {
	fsys := fstest.MapFS{"app.js": &fstest.MapFile{Data: []byte("new")}}

	dest := t.TempDir()
	writeFile := filepath.Join(dest, "app.js")
	if err := os.WriteFile(writeFile, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if err := CopyFS(fsys, dest); err != nil {
		t.Fatalf("CopyFS: %v", err)
	}

	data, err := os.ReadFile(writeFile)
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("app.js = %q, want overwrite with %q", data, "new")
	}
}
