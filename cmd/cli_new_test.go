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
