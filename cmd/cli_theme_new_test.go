package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeNewScaffoldsModule(t *testing.T) {
	useTempWorkingDir(t)

	oldVersion := Version
	Version = "v0.15.0"
	t.Cleanup(func() { Version = oldVersion })

	if err := newThemeProject("github.com/example/airway-sunset-theme", false, ""); err != nil {
		t.Fatalf("theme:new: %v", err)
	}

	dest := filepath.Join("airway-sunset-theme")
	for _, rel := range []string{
		"go.mod",
		"theme.go",
		"theme.templ",
		"theme_test.go",
		"assets.go",
		"generate.go",
		"assets/theme.css",
		"README.md",
		"README.zh-CN.md",
	} {
		if _, err := os.Stat(filepath.Join(dest, rel)); err != nil {
			t.Fatalf("expected scaffolded file %s: %v", rel, err)
		}
	}

	theme := readFile(t, filepath.Join(dest, "theme.go"))
	if !strings.Contains(theme, "package sunset") {
		t.Fatalf("expected derived package sunset, got:\n%s", theme)
	}
	if strings.Contains(theme, "{{package}}") || strings.Contains(theme, "{{module}}") {
		t.Fatalf("expected placeholders to be replaced, got:\n%s", theme)
	}

	goMod := readFile(t, filepath.Join(dest, "go.mod"))
	if !strings.Contains(goMod, "module github.com/example/airway-sunset-theme") {
		t.Fatalf("expected module line in go.mod, got:\n%s", goMod)
	}
	// Non-local scaffolds pin the framework at the CLI's own version and the
	// templ generation the framework ships with.
	if !strings.Contains(goMod, "github.com/daqing/airway v0.15.0") {
		t.Fatalf("expected pinned framework require in go.mod, got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "github.com/a-h/templ v0.3.1020") {
		t.Fatalf("expected pinned templ require in go.mod, got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "tool github.com/a-h/templ/cmd/templ") {
		t.Fatalf("expected templ tool directive in go.mod, got:\n%s", goMod)
	}
	if strings.Contains(goMod, "replace") {
		t.Fatalf("expected no replace outside a local checkout, got:\n%s", goMod)
	}
}

func TestThemeNewLocalCheckout(t *testing.T) {
	useTempWorkingDir(t)

	checkout := filepath.Join(t.TempDir(), "airway-src")
	makeDirs(t, checkout)
	writeFile(t, filepath.Join(checkout, "go.mod"), "module github.com/daqing/airway\n")
	writeFile(t, filepath.Join(checkout, "VERSION"), "0.9.3\n")

	if err := newThemeProject("sunset-theme", false, checkout); err != nil {
		t.Fatalf("theme:new local: %v", err)
	}

	goMod := readFile(t, filepath.Join("sunset-theme", "go.mod"))
	if !strings.Contains(goMod, "replace github.com/daqing/airway => "+checkout) {
		t.Fatalf("expected framework replace in go.mod, got:\n%s", goMod)
	}
	if strings.Contains(goMod, "require github.com/daqing/airway") {
		t.Fatalf("expected no framework pin alongside a local replace, got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "module sunset-theme") {
		t.Fatalf("expected bare module path in go.mod, got:\n%s", goMod)
	}
}

func TestThemeNewDirRef(t *testing.T) {
	useTempWorkingDir(t)

	if err := newThemeProject("./mytheme", false, ""); err != nil {
		t.Fatalf("theme:new ./mytheme: %v", err)
	}

	theme := readFile(t, filepath.Join("mytheme", "theme.go"))
	if !strings.Contains(theme, "package mytheme") {
		t.Fatalf("expected package mytheme, got:\n%s", theme)
	}
}

func TestThemePackageFromDir(t *testing.T) {
	for _, tc := range []struct {
		dir  string
		want string
	}{
		{"airway-terminal-theme", "terminal"},
		{"sunset", "sunset"},
		{"sunset-theme", "sunset"},
		{"airway-sunset-theme", "sunset"},
		{"my_theme.v2", "mythemev2"},
	} {
		if got := themePackageFromDir(tc.dir); got != tc.want {
			t.Fatalf("themePackageFromDir(%q) = %q, want %q", tc.dir, got, tc.want)
		}
	}
}

func TestThemeNewValidation(t *testing.T) {
	t.Run("undeducible package name", func(t *testing.T) {
		useTempWorkingDir(t)

		// A module segment that cannot start a Go package name.
		err := newThemeProject("github.com/example/123-theme", false, "")
		if err == nil || !strings.Contains(err.Error(), "cannot derive a Go package name") {
			t.Fatalf("theme:new 123-theme = %v, want package-derivation error", err)
		}
	})

	t.Run("non-empty target", func(t *testing.T) {
		useTempWorkingDir(t)
		makeDirs(t, "occupied")
		writeFile(t, filepath.Join("occupied", "keep.txt"), "x")

		if err := newThemeProject("occupied", false, ""); err == nil {
			t.Fatalf("expected non-empty target directory to fail")
		}
	})

	t.Run("argument count", func(t *testing.T) {
		useTempWorkingDir(t)

		if err := run([]string{"theme:new"}); err == nil {
			t.Fatalf("expected missing argument to fail")
		}
		if err := run([]string{"theme:new", "a", "b"}); err == nil {
			t.Fatalf("expected extra arguments to fail")
		}
		if err := run([]string{"theme:new", "-h"}); err != nil {
			t.Fatalf("theme:new -h: %v", err)
		}
	})
}
