package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadStoresFileAtSourcePathKey(t *testing.T) {
	storageRoot := t.TempDir()
	sourceRoot := t.TempDir()
	sourcePath := filepath.Join(sourceRoot, "images", "foo.png")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	content := []byte("png content")
	if err := os.WriteFile(sourcePath, content, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	t.Setenv("STORAGE_DRIVER", "local")
	t.Setenv("STORAGE_ROOT", storageRoot)

	if err := run([]string{"cli", "upload", sourcePath}); err != nil {
		t.Fatalf("run upload: %v", err)
	}

	key := strings.TrimLeft(filepath.ToSlash(filepath.Clean(sourcePath)), "/")
	storedPath := filepath.Join(storageRoot, filepath.FromSlash(key))
	got, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("unexpected uploaded content: %q", got)
	}
}

func TestUploadRejectsDirectory(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "local")
	t.Setenv("STORAGE_ROOT", t.TempDir())

	if err := runUpload([]string{t.TempDir()}); err == nil {
		t.Fatal("expected directory upload to fail")
	}
}

func TestUploadStoresFileAtExplicitKey(t *testing.T) {
	storageRoot := t.TempDir()
	sourcePath := filepath.Join(t.TempDir(), "foo.txt")
	if err := os.WriteFile(sourcePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	t.Setenv("STORAGE_DRIVER", "local")
	t.Setenv("STORAGE_ROOT", storageRoot)

	if err := run([]string{"cli", "upload", "assets/foo.txt", sourcePath}); err != nil {
		t.Fatalf("run upload: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(storageRoot, "assets", "foo.txt"))
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("unexpected uploaded content: %q", got)
	}
}

func TestUploadRequiresOnePath(t *testing.T) {
	for _, args := range [][]string{nil, {"one", "two", "three"}} {
		if err := runUpload(args); err == nil {
			t.Fatalf("expected args %q to fail", args)
		}
	}

	if err := runUpload([]string{"--help"}); err != nil {
		t.Fatalf("help should succeed: %v", err)
	}
}
