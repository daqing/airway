package clitemplate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldWritesTemplateWithModuleReplaced(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "demo")

	if err := Scaffold(destDir, "github.com/example/demo"); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	for _, rel := range []string{
		"go.mod",
		"main.go",
		"app.go",
		"VERSION",
		"version.go",
		".env.example",
		".gitignore",
		"config/routes.go",
		"app/api/health_api/routes.go",
		"app/models/user.go",
		"app/views/home/index.templ",
		"app/views/home/index_templ.go",
		"db/migrate/.keep",
	} {
		if _, err := os.Stat(filepath.Join(destDir, rel)); err != nil {
			t.Fatalf("expected scaffolded file %s: %v", rel, err)
		}
	}

	goMod := readFile(t, filepath.Join(destDir, "go.mod"))
	if !strings.Contains(goMod, "module github.com/example/demo") {
		t.Fatalf("expected module path in go.mod, got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "tool github.com/a-h/templ/cmd/templ") {
		t.Fatalf("expected templ tool directive in go.mod, got:\n%s", goMod)
	}

	routes := readFile(t, filepath.Join(destDir, "config", "routes.go"))
	if !strings.Contains(routes, `"github.com/example/demo/app/api/home_api"`) {
		t.Fatalf("expected project import rewritten, got:\n%s", routes)
	}
	if !strings.Contains(routes, `"github.com/daqing/airway/lib/engine"`) {
		t.Fatalf("expected framework imports untouched, got:\n%s", routes)
	}
}

func TestScaffoldLeavesNoPlaceholdersBehind(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "demo")

	if err := Scaffold(destDir, "myapp"); err != nil {
		t.Fatalf("scaffold: %v", err)
	}

	err := filepath.WalkDir(destDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		if strings.HasSuffix(path, ".tmpl") {
			t.Fatalf("template suffix not stripped: %s", path)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), ModulePlaceholder) {
			t.Fatalf("unresolved placeholder in %s", path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk scaffolded project: %v", err)
	}
}

func TestScaffoldEngineWritesTemplateWithPlaceholdersReplaced(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "airway-im-engine")

	if err := ScaffoldEngine(destDir, "github.com/example/airway-im-engine", "im"); err != nil {
		t.Fatalf("scaffold engine: %v", err)
	}

	for _, rel := range []string{
		"go.mod",
		"engine.go",
		"README.md",
		"app/api/im_api/routes.go",
		"app/api/im_api/index_action.go",
		"app/models/.keep",
		"db/migrate/.keep",
	} {
		if _, err := os.Stat(filepath.Join(destDir, rel)); err != nil {
			t.Fatalf("expected scaffolded file %s: %v", rel, err)
		}
	}

	goMod := readFile(t, filepath.Join(destDir, "go.mod"))
	if !strings.Contains(goMod, "module github.com/example/airway-im-engine") {
		t.Fatalf("expected module path in go.mod, got:\n%s", goMod)
	}

	engineFile := readFile(t, filepath.Join(destDir, "engine.go"))
	if !strings.Contains(engineFile, `"github.com/example/airway-im-engine/app/api/im_api"`) {
		t.Fatalf("expected engine import rewritten, got:\n%s", engineFile)
	}
	if !strings.Contains(engineFile, `{ return "im" }`) {
		t.Fatalf("expected engine name replaced, got:\n%s", engineFile)
	}

	err := filepath.WalkDir(destDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}

		if strings.HasSuffix(path, ".tmpl") {
			t.Fatalf("template suffix not stripped: %s", path)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), ModulePlaceholder) || strings.Contains(string(data), EnginePlaceholder) {
			t.Fatalf("unresolved placeholder in %s", path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk scaffolded engine: %v", err)
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
