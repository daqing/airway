package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// generateTemplViews runs the templ generator over app/views — the first
// half of `airway templates:compile`, shared with static:build/static:serve
// so a stale view cannot silently reach an export. Projects without the
// templ tool directive (or without app/views) report the tool error, which
// callers either tolerate or surface.
func generateTemplViews() error {
	gen := exec.Command("go", "tool", "templ", "generate", "-path", filepath.Join("app", "views"))
	gen.Stdout = os.Stdout
	gen.Stderr = os.Stderr
	return gen.Run()
}

// compileViewsOrNote regenerates the templ views right after a generator
// wrote new ones. `go run .` cannot rebuild a project whose views are not
// compiled yet (the root package imports them through export_<resource>.go),
// so without this the very next command the user tries would die in the Go
// toolchain with a build error — before the CLI can run at all. When templ
// cannot be invoked (e.g. a project without the tool directive), fall back
// to the manual path: the globally installed `airway templates:compile` runs
// against the sources without building the project.
func compileViewsOrNote() {
	if err := generateTemplViews(); err != nil {
		fmt.Println("\nNote: could not compile the .templ views automatically.")
		fmt.Println("Until they are compiled, `go run .` cannot rebuild this project.")
		fmt.Println("Run `airway templates:compile` with the globally installed")
		fmt.Println("airway — it compiles the views without building the project.")
	}
}

// runTemplatesCompile regenerates the committed Go code for the templ views.
// It runs `go tool templ generate` first: right after scaffolding, API
// actions import app/views packages whose only files are .templ, so
// `go generate ./...` cannot load those packages to find its directives —
// templ must run before the project compiles again. The trailing
// `go generate ./...` remains authoritative for non-templ generators.
// Extra arguments are forwarded as package patterns (default ./...).
func runTemplatesCompile(args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		fmt.Println("usage: airway templates:compile [packages]   # regenerate the templ views, then `go generate ./...`")
		return nil
	}

	packages := args
	if len(packages) == 0 {
		packages = []string{"./..."}
	}

	if err := generateTemplViews(); err != nil {
		// Projects without the templ tool directive (or without app/views)
		// still get the `go generate ./...` pass below, the mechanism that
		// worked before the tool directive existed.
		fmt.Fprintf(os.Stderr, "go tool templ generate: %v\n", err)
	}

	generate := exec.Command("go", append([]string{"generate"}, packages...)...)
	generate.Stdout = os.Stdout
	generate.Stderr = os.Stderr
	if err := generate.Run(); err != nil {
		return fmt.Errorf("go generate %s: %w", strings.Join(packages, " "), err)
	}
	return nil
}
