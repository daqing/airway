package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runCLIThemeInstall(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIThemeInstallUsage(os.Stdout)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: airway theme:install <module[@version] | /path/to/theme>")
	}

	arg := strings.TrimSpace(args[0])
	if arg == "" {
		return fmt.Errorf("usage: airway theme:install <module[@version] | /path/to/theme>")
	}

	if !hostProjectAt(".") {
		return fmt.Errorf("theme:install must run inside a host project (a directory with a main.go and a go.mod requiring github.com/daqing/airway)")
	}

	if info, err := os.Stat(arg); err == nil && info.IsDir() {
		return installLocalTheme(arg)
	}

	return installModuleTheme(arg)
}

// installLocalTheme wires a theme from a local directory into the host
// project: a replace directive pointing at the directory plus the require
// that `go get` records. Nothing is copied — themes are consumed as Go
// module dependencies, exactly like plugins but without the blank import.
func installLocalTheme(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	module, err := modulePathAt(abs)
	if err != nil {
		return fmt.Errorf("%s is not a Go module (go.mod: %v)", abs, err)
	}
	if module == airwayModulePath {
		return fmt.Errorf("%s is the airway framework itself, not a site theme", abs)
	}
	if _, ok := goModAirwayVersion(abs); !ok {
		return fmt.Errorf("%s does not look like an airway theme: its go.mod does not require github.com/daqing/airway", abs)
	}

	if err := runScaffoldCommand(".", scaffoldCommandTimeout, false,
		"go", "mod", "edit", "-replace", module+"="+abs); err != nil {
		return fmt.Errorf("add replace for %s: %w", module, err)
	}
	if err := runScaffoldCommand(".", scaffoldCommandTimeout, false, "go", "get", module); err != nil {
		return fmt.Errorf("go get %s: %w", module, err)
	}

	return printThemeNextSteps(module)
}

// installModuleTheme installs a published theme module with `go get`; the
// argument may carry an @version suffix.
func installModuleTheme(arg string) error {
	if err := runScaffoldCommand(".", scaffoldCommandTimeout, false, "go", "get", arg); err != nil {
		return fmt.Errorf("go get %s: %w", arg, err)
	}

	return printThemeNextSteps(moduleFromArg(arg))
}

// moduleFromArg strips an @version suffix from a module argument.
func moduleFromArg(arg string) string {
	if i := strings.Index(arg, "@"); i >= 0 {
		return arg[:i]
	}
	return arg
}

// printThemeNextSteps resolves the theme's package name and prints the
// ssg.go wiring for it.
func printThemeNextSteps(module string) error {
	pkg, err := themeCommandOutput("go", "list", "-f", "{{.Name}}", module)
	if err != nil {
		fmt.Printf("Installed %s\n", module)
		fmt.Printf("warning: could not resolve the theme's package name (%v)\n", err)
		fmt.Println("Add `s.Use(<package>.Theme)` to the site builder in ssg.go.")
		return nil
	}

	fmt.Printf("Installed %s (package %s)\n", module, pkg)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  # wire the theme into ssg.go:")
	fmt.Printf("  import %q\n", module)
	fmt.Println()
	fmt.Printf("  s.Use(%s.Theme)    // replaces the previous s.Use(...) call\n", pkg)
	fmt.Println()
	fmt.Println("  airway ssg:build           # export the static site into dist/")
	return nil
}

// themeCommandOutput runs a go command in the current directory and returns
// its trimmed stdout, with the same timeout and prompt guards as the
// scaffold commands.
func themeCommandOutput(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), scaffoldCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("`%s %s` timed out after %s", name, strings.Join(args, " "), scaffoldCommandTimeout)
		}
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

func printCLIThemeInstallUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway theme:install <module[@version] | /path/to/theme>")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "examples:")
	_, _ = fmt.Fprintln(w, "  airway theme:install github.com/daqing/airway-terminal-theme")
	_, _ = fmt.Fprintln(w, "  airway theme:install github.com/example/theme@v1.2.3")
	_, _ = fmt.Fprintln(w, "  airway theme:install ~/src/airway-terminal-theme")
}
