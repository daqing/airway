package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

	templGen := exec.Command("go", "tool", "templ", "generate", "-path", filepath.Join("app", "views"))
	templGen.Stdout = os.Stdout
	templGen.Stderr = os.Stderr
	if err := templGen.Run(); err != nil {
		// Projects without the templ tool directive (or without app/views)
		// fall back to plain `go generate ./...`, the mechanism that worked
		// before the tool directive existed.
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
