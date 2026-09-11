package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewEngineScaffoldsModule(t *testing.T) {
	wd := useTempWorkingDir(t)

	if err := newEngine("im", false); err != nil {
		t.Fatalf("new engine: %v", err)
	}

	engineFile := readFile(t, filepath.Join(wd, "im", "engine.go"))
	if !strings.Contains(engineFile, "package imengine") {
		t.Fatalf("expected imengine package, got:\n%s", engineFile)
	}
	if !strings.Contains(engineFile, `func (Engine) Name() string      { return "im" }`) {
		t.Fatalf("expected engine name in engine.go, got:\n%s", engineFile)
	}
	if !strings.Contains(engineFile, `"im/app/api/im_api"`) {
		t.Fatalf("expected engine api import, got:\n%s", engineFile)
	}

	goMod := readFile(t, filepath.Join(wd, "im", "go.mod"))
	if !strings.Contains(goMod, "module im") {
		t.Fatalf("expected module path in go.mod, got:\n%s", goMod)
	}

	routes := readFile(t, filepath.Join(wd, "im", "app", "api", "im_api", "routes.go"))
	if !strings.Contains(routes, "package im_api") {
		t.Fatalf("expected im_api package, got:\n%s", routes)
	}
}

func TestNewEngineDerivesNameFromModulePath(t *testing.T) {
	wd := useTempWorkingDir(t)

	if err := newEngine("github.com/example/airway-billing-engine", false); err != nil {
		t.Fatalf("new engine: %v", err)
	}

	engineFile := readFile(t, filepath.Join(wd, "airway-billing-engine", "engine.go"))
	if !strings.Contains(engineFile, `{ return "billing" }`) {
		t.Fatalf("expected derived engine name, got:\n%s", engineFile)
	}
	if !strings.Contains(engineFile, `"github.com/example/airway-billing-engine/app/api/billing_api"`) {
		t.Fatalf("expected module import rewritten, got:\n%s", engineFile)
	}
}

func TestNewEngineRejectsExistingNonEmptyDirectory(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "im"))
	writeFile(t, filepath.Join(wd, "im", "existing.txt"), "occupied\n")

	if err := newEngine("im", false); err == nil {
		t.Fatalf("expected error for non-empty directory")
	}
}

func TestNewEngineRejectsInvalidModulePath(t *testing.T) {
	useTempWorkingDir(t)

	for _, module := range []string{"", "/leading-slash", "UPPER Case", "has space"} {
		if err := newEngine(module, false); err == nil {
			t.Fatalf("expected error for module path %q", module)
		}
	}
}

func TestRunEngineNewHelpPrintsUsage(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"engine", "new", "-h"}); err != nil {
			t.Fatalf("run engine new help: %v", err)
		}
	})

	if !strings.Contains(output, "airway engine new <module-path>") {
		t.Fatalf("expected engine new usage output, got:\n%s", output)
	}
}
