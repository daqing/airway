package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSGNewScaffoldsProject(t *testing.T) {
	wd := useTempWorkingDir(t)

	oldVersion := Version
	Version = "v0.15.0"
	t.Cleanup(func() { Version = oldVersion })

	if err := newSSGProject("mysite", false, ""); err != nil {
		t.Fatalf("ssg:new: %v", err)
	}

	for _, rel := range []string{
		"mysite/go.mod",
		"mysite/main.go",
		"mysite/ssg.go",
		"mysite/version.go",
		"mysite/VERSION",
		"mysite/.gitignore",
		"mysite/README.md",
		"mysite/README.zh-CN.md",
	} {
		if _, err := os.Stat(filepath.Join(wd, rel)); err != nil {
			t.Fatalf("expected scaffolded file %s: %v", rel, err)
		}
	}

	siteFile := readFile(t, filepath.Join(wd, "mysite", "ssg.go"))
	if strings.Contains(siteFile, "{{module}}") {
		t.Fatalf("expected module placeholder to be replaced, got:\n%s", siteFile)
	}
	if !strings.Contains(siteFile, `"mysite"`) {
		t.Fatalf("expected site title to use the module name, got:\n%s", siteFile)
	}
	if !strings.Contains(siteFile, `"github.com/daqing/airway/themes/corporate"`) {
		t.Fatalf("expected the corporate theme import, got:\n%s", siteFile)
	}

	// Non-local scaffolds pin both modules at the CLI's own version so the
	// first tidy pulls known versions. The bare substring matches both the
	// single-line and require-block go.mod layouts.
	goMod := readFile(t, filepath.Join(wd, "mysite", "go.mod"))
	if !strings.Contains(goMod, "github.com/daqing/airway v0.15.0") {
		t.Fatalf("expected pinned framework require in go.mod, got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "github.com/daqing/airway/themes/corporate v0.15.0") {
		t.Fatalf("expected pinned theme require in go.mod, got:\n%s", goMod)
	}
}

func TestSSGNewReplacesFrameworkAndThemeWithLocalCheckout(t *testing.T) {
	wd := useTempWorkingDir(t)

	checkout := filepath.Join(wd, "airway-src")
	makeDirs(t, checkout)
	writeFile(t, filepath.Join(checkout, "go.mod"), "module github.com/daqing/airway\n")
	writeFile(t, filepath.Join(checkout, "VERSION"), "0.9.3\n")

	if err := newSSGProject("localdev", false, checkout); err != nil {
		t.Fatalf("ssg:new local: %v", err)
	}

	goMod := readFile(t, filepath.Join(wd, "localdev", "go.mod"))
	if !strings.Contains(goMod, "replace github.com/daqing/airway => "+checkout) {
		t.Fatalf("expected framework replace in go.mod, got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "replace github.com/daqing/airway/themes/corporate => "+filepath.Join(checkout, "themes", "corporate")) {
		t.Fatalf("expected theme replace in go.mod, got:\n%s", goMod)
	}
	if strings.Contains(goMod, "require") {
		t.Fatalf("expected no version pins alongside local replaces, got:\n%s", goMod)
	}
}

func TestSSGNewRejectsNonEmptyDirAndBadArgs(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "occupied"))
	writeFile(t, filepath.Join(wd, "occupied", "keep.txt"), "x")

	if err := newSSGProject("occupied", false, ""); err == nil {
		t.Fatalf("expected non-empty target directory to fail")
	}

	if err := run([]string{"ssg:new"}); err == nil {
		t.Fatalf("expected missing argument to fail")
	}

	if err := run([]string{"ssg:new", "a", "b"}); err == nil {
		t.Fatalf("expected extra arguments to fail")
	}

	if err := run([]string{"ssg:new", "-h"}); err != nil {
		t.Fatalf("ssg:new -h: %v", err)
	}
}
