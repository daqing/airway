package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProjectScaffoldsModule(t *testing.T) {
	wd := useTempWorkingDir(t)

	if err := newProject("github.com/example/demo", false); err != nil {
		t.Fatalf("new project: %v", err)
	}

	mainFile := readFile(t, filepath.Join(wd, "demo", "main.go"))
	if !strings.Contains(mainFile, "runServer()") {
		t.Fatalf("expected server entry in main.go, got:\n%s", mainFile)
	}

	goMod := readFile(t, filepath.Join(wd, "demo", "go.mod"))
	if !strings.Contains(goMod, "module github.com/example/demo") {
		t.Fatalf("expected module path in go.mod, got:\n%s", goMod)
	}

	for _, rel := range []string{
		".env.example",
		"Dockerfile",
		"app/views/home/index.templ",
		"app/views/home/index_templ.go",
	} {
		content := readFile(t, filepath.Join(wd, "demo", rel))
		if strings.Contains(content, "1900") {
			t.Fatalf("expected no leftover port 1900 in scaffolded %s", rel)
		}
		if !strings.Contains(content, "1905") {
			t.Fatalf("expected scaffolded %s to use port 1905", rel)
		}
	}

	envExample := readFile(t, filepath.Join(wd, "demo", ".env.example"))
	if env := readFile(t, filepath.Join(wd, "demo", ".env")); env != envExample {
		t.Fatalf("expected .env seeded from .env.example, got:\n%s", env)
	}
}

func TestNewProjectRejectsExistingNonEmptyDirectory(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "demo"))
	writeFile(t, filepath.Join(wd, "demo", "existing.txt"), "occupied\n")

	if err := newProject("demo", false); err == nil {
		t.Fatalf("expected error for non-empty directory")
	}
}

func TestNewProjectRejectsInvalidModulePath(t *testing.T) {
	useTempWorkingDir(t)

	for _, module := range []string{"", "/leading-slash", "UPPER Case", "has space"} {
		if err := newProject(module, false); err == nil {
			t.Fatalf("expected error for module path %q", module)
		}
	}
}

func TestRunNewHelpPrintsUsage(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"new", "-h"}); err != nil {
			t.Fatalf("run new help: %v", err)
		}
	})

	if !strings.Contains(output, "airway new <module-path>") {
		t.Fatalf("expected new usage output, got:\n%s", output)
	}
}
