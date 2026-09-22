package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplatesCompileHelp(t *testing.T) {
	output := captureStdout(t, func() {
		if err := runTemplatesCompile([]string{"-h"}); err != nil {
			t.Fatalf("runTemplatesCompile help: %v", err)
		}
	})

	if !strings.Contains(output, "usage: airway templates:compile") {
		t.Fatalf("expected usage output, got:\n%s", output)
	}
}

func TestTemplatesCompileRunsGoGenerate(t *testing.T) {
	wd := useTempWorkingDir(t)

	writeFile := func(path, content string) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(filepath.Join(wd, "go.mod"), "module tmpcompile\n\ngo 1.25\n")
	writeFile(filepath.Join(wd, "doc.go"), "package tmpcompile\n\n//go:generate go run ./gen\n")
	makeDirs(t, filepath.Join(wd, "gen"))
	writeFile(filepath.Join(wd, "gen", "main.go"), `package main

import "os"

func main() {
	if err := os.WriteFile("compiled.txt", []byte("ok"), 0o644); err != nil {
		panic(err)
	}
}
`)

	if err := runTemplatesCompile(nil); err != nil {
		t.Fatalf("runTemplatesCompile: %v", err)
	}

	if _, err := os.Stat(filepath.Join(wd, "compiled.txt")); err != nil {
		t.Fatalf("go generate did not run the directive: %v", err)
	}
}

func TestTemplatesCompileOutsideModule(t *testing.T) {
	useTempWorkingDir(t)

	if err := runTemplatesCompile(nil); err == nil {
		t.Fatalf("expected error when running outside a Go module")
	}
}

func TestTemplatesCompileDispatch(t *testing.T) {
	useTempWorkingDir(t)

	output := captureStdout(t, func() {
		if err := run([]string{"templates:compile", "-h"}); err != nil {
			t.Fatalf("run templates:compile help: %v", err)
		}
	})

	if !strings.Contains(output, "usage: airway templates:compile") {
		t.Fatalf("expected usage output through dispatch, got:\n%s", output)
	}
}
