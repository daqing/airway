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
		"VERSION",
		"version.go",
		"export.go",
		".env.example",
		".gitignore",
		"config/routes.go",
		"app/api/health_api/routes.go",
		"app/models/registry.go",
		"app/views/home/index.templ",
		"app/views/home/index_templ.go",
		"db/migrate/.keep",
		"deps/.keep",
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
	if !strings.Contains(routes, `"github.com/daqing/airway/lib/plugin"`) {
		t.Fatalf("expected framework imports untouched, got:\n%s", routes)
	}

	// The web entry must mount the project's own routes (lib/app requires it).
	mainGo := readFile(t, filepath.Join(destDir, "main.go"))
	if !strings.Contains(mainGo, "app.WithRoutes(config.Routes, config.HealthRoutes)") {
		t.Fatalf("expected main.go to mount project routes, got:\n%s", mainGo)
	}

	// The scaffold must ship the LISTEN address, not the legacy PORT key.
	envExample := readFile(t, filepath.Join(destDir, ".env.example"))
	if !strings.Contains(envExample, `LISTEN="`) {
		t.Fatalf("expected LISTEN in .env.example, got:\n%s", envExample)
	}
	if strings.Contains(envExample, "PORT=") {
		t.Fatalf("expected no legacy PORT in .env.example, got:\n%s", envExample)
	}

	exportGo := readFile(t, filepath.Join(destDir, "export.go"))
	if !strings.Contains(exportGo, `"github.com/example/demo/app/views/home"`) {
		t.Fatalf("expected project import rewritten in export.go, got:\n%s", exportGo)
	}
	if !strings.Contains(exportGo, "static.Page{Slug: \"/\", Component: home.Index()}") {
		t.Fatalf("expected welcome page registered in export.go, got:\n%s", exportGo)
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

func TestScaffoldPluginWritesTemplateWithPlaceholdersReplaced(t *testing.T) {
	destDir := filepath.Join(t.TempDir(), "airway-im-plugin")

	if err := ScaffoldPlugin(destDir, "github.com/example/airway-im-plugin", "im"); err != nil {
		t.Fatalf("scaffold plugin: %v", err)
	}

	for _, rel := range []string{
		"go.mod",
		"plugin.go",
		"README.md",
		".gitignore",
		"install/lib/api/im_api/routes.go",
		"install/lib/api/im_api/index_action.go",
		"install/lib/models/.keep",
		"install/host/db/migrate/.keep",
		"install/deps/.keep",
		"install/ignore/.keep",
	} {
		if _, err := os.Stat(filepath.Join(destDir, rel)); err != nil {
			t.Fatalf("expected scaffolded file %s: %v", rel, err)
		}
	}

	goMod := readFile(t, filepath.Join(destDir, "go.mod"))
	if !strings.Contains(goMod, "module github.com/example/airway-im-plugin") {
		t.Fatalf("expected module path in go.mod, got:\n%s", goMod)
	}

	pluginFile := readFile(t, filepath.Join(destDir, "plugin.go"))
	if !strings.Contains(pluginFile, `"github.com/example/airway-im-plugin/install/lib/api/im_api"`) {
		t.Fatalf("expected plugin import rewritten, got:\n%s", pluginFile)
	}
	if !strings.Contains(pluginFile, `{ return "im" }`) {
		t.Fatalf("expected plugin name replaced, got:\n%s", pluginFile)
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
		if strings.Contains(string(data), ModulePlaceholder) || strings.Contains(string(data), PluginPlaceholder) {
			t.Fatalf("unresolved placeholder in %s", path)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk scaffolded plugin: %v", err)
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
