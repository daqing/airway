package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCLIEnvLoadsDotEnvWhenPresent(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	if err := os.Unsetenv("AIRWAY_DB_DSN"); err != nil {
		t.Fatalf("unset AIRWAY_DB_DSN: %v", err)
	}

	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte("AIRWAY_DB_DSN=sqlite://./tmp/test.db\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	loadCLIEnv()

	if got := os.Getenv("AIRWAY_DB_DSN"); got != "sqlite://./tmp/test.db" {
		t.Fatalf("expected AIRWAY_DB_DSN from .env, got %q", got)
	}
}

func TestLoadCLIEnvDoesNotFailWhenDotEnvMissing(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	if err := os.Unsetenv("AIRWAY_DB_DSN"); err != nil {
		t.Fatalf("unset AIRWAY_DB_DSN: %v", err)
	}

	loadCLIEnv()

	if got := os.Getenv("AIRWAY_DB_DSN"); got != "" {
		t.Fatalf("expected AIRWAY_DB_DSN to remain empty, got %q", got)
	}
}
