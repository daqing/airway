package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

// setupThemeProject builds a host project (go.mod + main.go) alongside a
// stub airway checkout and a local theme module, all resolvable without
// network access.
func setupThemeProject(t *testing.T) (themeDir string) {
	t.Helper()

	wd := useTempWorkingDir(t)

	stub := filepath.Join(wd, "airway-stub")
	makeDirs(t, stub)
	writeFile(t, filepath.Join(stub, "go.mod"), "module github.com/daqing/airway\n")

	writeFile(t, filepath.Join(wd, "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(wd, "go.mod"), `module github.com/example/demo

go 1.27

require github.com/daqing/airway v0.0.0

replace github.com/daqing/airway => `+stub+`
`)

	themeDir = filepath.Join(wd, "theme")
	makeDirs(t, themeDir)
	writeFile(t, filepath.Join(themeDir, "go.mod"), `module github.com/example/airway-theme-fixture

go 1.27

require github.com/daqing/airway v0.0.0
`)
	writeFile(t, filepath.Join(themeDir, "theme.go"), "package fixture\n")

	// theme:install resolves the directory through filepath.Abs, which
	// canonicalizes macOS /var symlinks to /private/var.
	if resolved, err := filepath.EvalSymlinks(themeDir); err == nil {
		themeDir = resolved
	}

	return themeDir
}

func TestThemeInstallLocalDirectory(t *testing.T) {
	themeDir := setupThemeProject(t)

	output := captureStdout(t, func() {
		if runErr := run([]string{"theme:install", "theme"}); runErr != nil {
			t.Fatalf("theme:install: %v", runErr)
		}
	})

	goMod := readFile(t, "go.mod")
	if !strings.Contains(goMod, "replace github.com/example/airway-theme-fixture => "+themeDir) {
		t.Fatalf("expected theme replace in go.mod, got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "require github.com/example/airway-theme-fixture") {
		t.Fatalf("expected theme require in go.mod, got:\n%s", goMod)
	}

	for _, want := range []string{
		`import "github.com/example/airway-theme-fixture"`,
		"s.Use(fixture.Theme)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("next steps missing %q, got:\n%s", want, output)
		}
	}
}

func TestThemeInstallValidation(t *testing.T) {
	t.Run("outside host project", func(t *testing.T) {
		useTempWorkingDir(t)

		err := run([]string{"theme:install", "github.com/example/theme"})
		if err == nil || !strings.Contains(err.Error(), "host project") {
			t.Fatalf("theme:install outside a project = %v, want host-project error", err)
		}
	})

	t.Run("directory without go.mod", func(t *testing.T) {
		setupThemeProject(t)
		makeDirs(t, "plain-dir")

		err := run([]string{"theme:install", "plain-dir"})
		if err == nil || !strings.Contains(err.Error(), "not a Go module") {
			t.Fatalf("theme:install plain-dir = %v, want not-a-module error", err)
		}
	})

	t.Run("module without airway dependency", func(t *testing.T) {
		setupThemeProject(t)
		makeDirs(t, "notatheme")
		writeFile(t, filepath.Join("notatheme", "go.mod"), "module github.com/example/plain\ngo 1.27\n")

		err := run([]string{"theme:install", "notatheme"})
		if err == nil || !strings.Contains(err.Error(), "does not look like an airway theme") {
			t.Fatalf("theme:install notatheme = %v, want not-a-theme error", err)
		}
	})

	t.Run("airway framework itself", func(t *testing.T) {
		setupThemeProject(t)

		err := run([]string{"theme:install", "airway-stub"})
		if err == nil || !strings.Contains(err.Error(), "not a site theme") {
			t.Fatalf("theme:install airway-stub = %v, want framework error", err)
		}
	})

	t.Run("argument count", func(t *testing.T) {
		setupThemeProject(t)

		if err := run([]string{"theme:install"}); err == nil {
			t.Fatalf("expected missing argument to fail")
		}
		if err := run([]string{"theme:install", "a", "b"}); err == nil {
			t.Fatalf("expected extra arguments to fail")
		}
		if err := run([]string{"theme:install", "-h"}); err != nil {
			t.Fatalf("theme:install -h: %v", err)
		}
	})
}

func TestModuleFromArg(t *testing.T) {
	for _, tc := range []struct {
		arg  string
		want string
	}{
		{"github.com/example/theme", "github.com/example/theme"},
		{"github.com/example/theme@v1.2.3", "github.com/example/theme"},
		{"github.com/example/theme@latest", "github.com/example/theme"},
	} {
		if got := moduleFromArg(tc.arg); got != tc.want {
			t.Fatalf("moduleFromArg(%q) = %q, want %q", tc.arg, got, tc.want)
		}
	}
}
