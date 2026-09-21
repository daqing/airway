package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runTemplatesCompile regenerates the committed Go code for the templ views
// by proxying to `go generate ./...` — the generate.go at the project root
// declares the `go tool templ generate` directive. Extra arguments are
// forwarded as package patterns (default ./...).
func runTemplatesCompile(args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		fmt.Println("usage: airway templates:compile [packages]   # shorthand for `go generate ./...`")
		return nil
	}

	packages := args
	if len(packages) == 0 {
		packages = []string{"./..."}
	}

	generate := exec.Command("go", append([]string{"generate"}, packages...)...)
	generate.Stdout = os.Stdout
	generate.Stderr = os.Stderr
	if err := generate.Run(); err != nil {
		return fmt.Errorf("go generate %s: %w", strings.Join(packages, " "), err)
	}
	return nil
}
