package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	// Linking the framework's config registers the route source via init(),
	// mirroring what a project binary's own config package does.
	_ "github.com/daqing/airway/config"
)

func TestRunCLIOpenAPIGenerateWritesDocument(t *testing.T) {
	useTempWorkingDir(t)

	if err := run([]string{"cli", "openapi:generate"}); err != nil {
		t.Fatalf("run openapi:generate: %v", err)
	}

	content := readFile(t, filepath.Join(".", "openapi.json"))

	var doc map[string]any
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("unmarshal openapi.json: %v", err)
	}

	if doc["openapi"] != "3.2.0" {
		t.Fatalf("expected OpenAPI 3.2.0, got %v", doc["openapi"])
	}

	for _, path := range []string{"/health", "/api/v1/storage", "/api/v1/storage/{key}"} {
		if !strings.Contains(content, `"`+path+`"`) {
			t.Fatalf("expected %s in openapi.json, got:\n%s", path, content)
		}
	}

	if strings.Contains(content, `/assets/`) {
		t.Fatal("static asset routes must be excluded from the document")
	}
}

func TestRunCLIOpenAPIGenerateHonorsOutFlag(t *testing.T) {
	useTempWorkingDir(t)

	if err := run([]string{"cli", "openapi:generate", "--out", "docs/api.json"}); err != nil {
		t.Fatalf("run openapi:generate --out: %v", err)
	}

	content := readFile(t, filepath.Join(".", "docs", "api.json"))
	if !strings.Contains(content, `"openapi": "3.2.0"`) {
		t.Fatalf("expected OpenAPI 3.2.0 in docs/api.json, got:\n%s", content)
	}
}

func TestRunCLIOpenAPIGenerateHelpPrintsUsage(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"cli", "openapi:generate", "-h"}); err != nil {
			t.Fatalf("run openapi:generate help: %v", err)
		}
	})

	if !strings.Contains(output, "airway openapi:generate [--out path]") {
		t.Fatalf("expected usage output, got:\n%s", output)
	}
}
