package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewProjectScaffoldsModule(t *testing.T) {
	wd := useTempWorkingDir(t)

	if err := newProject("github.com/example/demo", false, false, ""); err != nil {
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

	if info, err := os.Stat(filepath.Join(wd, "demo", ".git")); err != nil || !info.IsDir() {
		t.Fatalf("expected a git repository in the scaffolded project: %v", err)
	}
}

func TestNewProjectPinsAirwayVersion(t *testing.T) {
	wd := useTempWorkingDir(t)

	oldVersion := Version
	Version = "v9.9.9"
	t.Cleanup(func() { Version = oldVersion })

	if err := newProject("pinned", false, false, ""); err != nil {
		t.Fatalf("new project: %v", err)
	}

	goMod := readFile(t, filepath.Join(wd, "pinned", "go.mod"))
	if !strings.Contains(goMod, "require github.com/daqing/airway v9.9.9") {
		t.Fatalf("expected pinned airway require in go.mod, got:\n%s", goMod)
	}
}

func TestNewProjectSkipsPinForDevBuild(t *testing.T) {
	wd := useTempWorkingDir(t)

	oldVersion := Version
	Version = "dev"
	t.Cleanup(func() { Version = oldVersion })

	if err := newProject("devbuild", false, false, ""); err != nil {
		t.Fatalf("new project: %v", err)
	}

	goMod := readFile(t, filepath.Join(wd, "devbuild", "go.mod"))
	if strings.Contains(goMod, "require github.com/daqing/airway") {
		t.Fatalf("expected no airway require for dev builds, got:\n%s", goMod)
	}
}

func TestNewProjectRejectsExistingNonEmptyDirectory(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "demo"))
	writeFile(t, filepath.Join(wd, "demo", "existing.txt"), "occupied\n")

	if err := newProject("demo", false, false, ""); err == nil {
		t.Fatalf("expected error for non-empty directory")
	}
}

func TestNewProjectFromAbsolutePath(t *testing.T) {
	wd := useTempWorkingDir(t)

	destDir := filepath.Join(wd, "nested", "foobar")
	if err := newProject(destDir, false, false, ""); err != nil {
		t.Fatalf("new project: %v", err)
	}

	goMod := readFile(t, filepath.Join(destDir, "go.mod"))
	if !strings.Contains(goMod, "module foobar") {
		t.Fatalf("expected module path from the last path segment, got:\n%s", goMod)
	}

	if _, err := os.Stat(filepath.Join(destDir, "main.go")); err != nil {
		t.Fatalf("expected project scaffolded at %s: %v", destDir, err)
	}
}

func TestNewProjectRejectsInvalidModulePath(t *testing.T) {
	useTempWorkingDir(t)

	for _, module := range []string{"", "UPPER Case", "has space", "/tmp/UPPER Case"} {
		if err := newProject(module, false, false, ""); err == nil {
			t.Fatalf("expected error for module path %q", module)
		}
	}
}

func TestRunScaffoldCommandTimesOut(t *testing.T) {
	err := runScaffoldCommand(t.TempDir(), 50*time.Millisecond, false, "sleep", "10")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestRunNewHelpPrintsUsage(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"new", "-h"}); err != nil {
			t.Fatalf("run new help: %v", err)
		}
	})

	if !strings.Contains(output, "airway new [--local[=path]] <module-path | directory>") {
		t.Fatalf("expected new usage output, got:\n%s", output)
	}
}

func TestResolveLocalCheckout(t *testing.T) {
	wd := useTempWorkingDir(t)
	checkout := filepath.Join(wd, "airway-src")
	makeDirs(t, checkout)
	writeFile(t, filepath.Join(checkout, "go.mod"), "module github.com/daqing/airway\n\ngo 1.26\n")

	if _, err := resolveLocalCheckout(filepath.Join(wd, "empty")); err == nil || !strings.Contains(err.Error(), "go.mod not found") {
		t.Fatalf("missing go.mod: err = %v, want go.mod not found", err)
	}

	writeFile(t, filepath.Join(wd, "go.mod"), "module github.com/example/other\n")
	if _, err := resolveLocalCheckout(wd); err == nil || !strings.Contains(err.Error(), "module is not github.com/daqing/airway") {
		t.Fatalf("foreign module: err = %v, want module mismatch", err)
	}

	if _, err := resolveLocalCheckout(checkout); err == nil || !strings.Contains(err.Error(), "VERSION not found") {
		t.Fatalf("missing VERSION: err = %v, want VERSION not found", err)
	}

	writeFile(t, filepath.Join(checkout, "VERSION"), "0.9.3\n")
	abs, err := resolveLocalCheckout(checkout)
	if err != nil {
		t.Fatalf("resolveLocalCheckout: %v", err)
	}
	if abs != checkout {
		t.Fatalf("resolveLocalCheckout = %q, want %q", abs, checkout)
	}
}

func TestNewProjectReplacesWithLocalCheckout(t *testing.T) {
	wd := useTempWorkingDir(t)

	oldVersion := Version
	Version = "v9.9.9"
	t.Cleanup(func() { Version = oldVersion })

	checkout := filepath.Join(wd, "airway-src")
	makeDirs(t, checkout)
	writeFile(t, filepath.Join(checkout, "go.mod"), "module github.com/daqing/airway\n")
	writeFile(t, filepath.Join(checkout, "VERSION"), "0.9.3\n")

	if err := newProject("localdev", false, false, checkout); err != nil {
		t.Fatalf("new project: %v", err)
	}

	goMod := readFile(t, filepath.Join(wd, "localdev", "go.mod"))
	if !strings.Contains(goMod, "replace github.com/daqing/airway => "+checkout) {
		t.Fatalf("expected local replace in go.mod, got:\n%s", goMod)
	}
	if strings.Contains(goMod, "require github.com/daqing/airway") {
		t.Fatalf("expected no pin alongside a local replace, got:\n%s", goMod)
	}
}

func TestImpliedLocalCheckout(t *testing.T) {
	useTempWorkingDir(t)

	if dir, ok := impliedLocalCheckout(); ok {
		t.Fatalf("impliedLocalCheckout outside a checkout = %q, want none", dir)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	checkout := filepath.Join(wd, "airway-src")
	makeDirs(t, checkout)
	writeFile(t, filepath.Join(checkout, "go.mod"), "module github.com/daqing/airway\n")
	writeFile(t, filepath.Join(checkout, "VERSION"), "0.9.3\n")
	if err := os.Chdir(checkout); err != nil {
		t.Fatal(err)
	}

	dir, ok := impliedLocalCheckout()
	if !ok || dir != checkout {
		t.Fatalf("impliedLocalCheckout inside checkout = %q, %v; want %q", dir, ok, checkout)
	}
}

func TestRunNewLocalFlagRejectsNonCheckout(t *testing.T) {
	useTempWorkingDir(t)

	err := run([]string{"new", "--local", "demo"})
	if err == nil || !strings.Contains(err.Error(), "--local") {
		t.Fatalf("run new --local outside a checkout = %v, want --local error", err)
	}

	err = run([]string{"new", "--local=", "demo"})
	if err == nil || !strings.Contains(err.Error(), "--local needs a path") {
		t.Fatalf("run new --local= = %v, want path-required error", err)
	}
}

func TestNewProjectRunsJsInstall(t *testing.T) {
	wd := useTempWorkingDir(t)

	// A closed registry port fails fast; the failed install must degrade to a
	// warning instead of aborting the scaffolded project.
	t.Setenv("AIRWAY_JS_REGISTRY", "http://127.0.0.1:1")

	output := captureStdout(t, func() {
		if err := newProject("jsfail", false, true, ""); err != nil {
			t.Fatalf("new project: %v", err)
		}
	})

	if !strings.Contains(output, "WARNING: `airway js:install` failed") {
		t.Fatalf("expected js:install warning, got:\n%s", output)
	}
	if !strings.Contains(output, "Next steps") || !strings.Contains(output, "airway js:install") {
		t.Fatalf("expected manual js:install hint in next steps, got:\n%s", output)
	}

	if _, err := os.Stat(filepath.Join(wd, "jsfail", "main.go")); err != nil {
		t.Fatalf("expected project scaffolded despite js:install failure: %v", err)
	}
}

func TestInstallScaffoldJSUsesCachedVendor(t *testing.T) {
	wd := useTempWorkingDir(t)

	root := filepath.Join(wd, "cached")
	makeDirs(t, root)
	writeFile(t, filepath.Join(root, "js.pkg.json"),
		`{"deps":{"preact":"10.29.8"},"lock":{"preact":{"version":"10.29.8"}}}`)
	vendor := filepath.Join(root, "app", "assets", "js", "vendor", "preact")
	makeDirs(t, vendor)
	writeFile(t, filepath.Join(vendor, "package.json"), `{"name":"preact","version":"10.29.8"}`)

	output := captureStdout(t, func() {
		if err := installScaffoldJS(root); err != nil {
			t.Fatalf("installScaffoldJS: %v", err)
		}
	})

	if !strings.Contains(output, "cached preact@10.29.8") {
		t.Fatalf("expected cached install without network, got:\n%s", output)
	}
}
