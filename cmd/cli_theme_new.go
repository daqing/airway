package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/daqing/airway/cmd/clitemplate"
)

// themeTemplVersion pins the templ runtime and tool in scaffolded theme
// modules to the generation the framework itself ships with, so generated
// views and the runtime stay in sync across a site's module graph.
const themeTemplVersion = "v0.3.1020"

func runCLIThemeNew(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIThemeNewUsage(os.Stdout)
		return nil
	}

	// --local[=path] points the scaffolded theme at an airway source
	// checkout through a replace directive, for theme development.
	local := ""
	localSet := false
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		switch {
		case arg == "--local" || arg == "-local":
			local, localSet = ".", true
		case strings.HasPrefix(arg, "--local="), strings.HasPrefix(arg, "-local="):
			local, localSet = strings.SplitN(arg, "=", 2)[1], true
		default:
			rest = append(rest, arg)
		}
	}

	if len(rest) != 1 {
		return fmt.Errorf("usage: airway theme:new [--local[=path]] <module-path | path>")
	}

	if localSet {
		if local == "" {
			return fmt.Errorf("--local needs a path to an airway checkout")
		}
		dir, err := resolveLocalCheckout(local)
		if err != nil {
			return err
		}
		local = dir
	} else if dir, ok := impliedLocalCheckout(); ok {
		local = dir
		fmt.Printf("Inside an airway checkout; implying --local=%s\n", dir)
	}

	return newThemeProject(strings.TrimSpace(rest[0]), true, local)
}

// newThemeProject scaffolds a site theme module. ref is either a module path
// — the theme is created under the working directory at its last segment —
// or a filesystem path, in which case the theme is created at that path and
// the module path is the path's last segment (same conventions as
// plugin:new).
func newThemeProject(ref string, tidy bool, localDir string) error {
	module := ref
	destDir := ""
	dirName := ref

	if isDirRef(ref) {
		destDir, _ = filepath.Abs(filepath.Clean(ref))
		dirName = filepath.Base(destDir)
		module = dirName
		if !modulePathPattern.MatchString(dirName) {
			return fmt.Errorf("invalid module path %q", ref)
		}
	} else {
		if !modulePathPattern.MatchString(ref) {
			return fmt.Errorf("invalid module path %q", ref)
		}
		if idx := strings.LastIndex(ref, "/"); idx >= 0 {
			dirName = ref[idx+1:]
		}
	}

	pkg := themePackageFromDir(dirName)
	if !pluginNamePattern.MatchString(pkg) {
		return fmt.Errorf("cannot derive a Go package name from %q; use a *-theme name with a distinct word (e.g. airway-sunset-theme)", ref)
	}

	if destDir == "" {
		destDir = filepath.Join(".", dirName)
	}

	if info, err := os.Stat(destDir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s already exists and is not a directory", destDir)
		}

		entries, err := os.ReadDir(destDir)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("directory %s already exists and is not empty", destDir)
		}
	}

	if err := clitemplate.ScaffoldTheme(destDir, module, pkg); err != nil {
		return fmt.Errorf("scaffold theme: %w", err)
	}

	fmt.Printf("Created a new Airway site theme in %s (module %s, package %s)\n", destDir, module, pkg)

	pinned := false

	if localDir != "" {
		if err := runScaffoldCommand(destDir, scaffoldCommandTimeout, false,
			"go", "mod", "edit", "-replace", airwayModulePath+"="+localDir); err != nil {
			return fmt.Errorf("replace airway with local checkout: %w", err)
		}
		fmt.Printf("Replaced the framework with the local checkout %s\n", localDir)
	} else if strings.HasPrefix(Version, "v") {
		if err := runScaffoldCommand(destDir, scaffoldCommandTimeout, false,
			"go", "mod", "edit", "-require", airwayModulePath+"@"+Version); err != nil {
			return fmt.Errorf("pin airway version in go.mod: %w", err)
		}
		pinned = true
	}

	if err := runScaffoldCommand(destDir, scaffoldCommandTimeout, false,
		"go", "mod", "edit", "-require", "github.com/a-h/templ@"+themeTemplVersion); err != nil {
		return fmt.Errorf("pin templ version in go.mod: %w", err)
	}

	if tidy {
		tidyErr := tidyScaffold(destDir)
		if tidyErr != nil && pinned && localDir == "" {
			// unpinScaffoldAirwayVersion strips the framework require that
			// does not resolve; tidy then discovers a published version.
			if unpinErr := unpinScaffoldAirwayVersion(destDir); unpinErr == nil {
				tidyErr = tidyScaffold(destDir)
			}
		}
		if tidyErr != nil {
			fmt.Printf("WARNING: `go mod tidy` failed: %v\nRun it manually inside %s before building.\n", tidyErr, destDir)
		}

		// Generate the templ views so the module compiles right away; this
		// needs the resolved module graph, hence after tidy.
		if err := runScaffoldCommand(destDir, scaffoldCommandTimeout, false,
			"go", "tool", "templ", "generate", "-path", "."); err != nil {
			fmt.Printf("WARNING: templ generate failed: %v\nRun `go generate ./...` manually inside %s before building.\n", err, destDir)
		}
	}

	if err := initScaffoldGit(destDir); err != nil {
		fmt.Printf("WARNING: `git init` failed: %v\n", err)
	}

	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", destDir)
	fmt.Println("  # edit theme.go, theme.templ, and assets/theme.css")
	fmt.Println("  go generate ./...           # regenerate the templ views")
	fmt.Println("  go test ./...")
	fmt.Printf("\nInstall it into a site project:\n  airway theme:install %s\n", module)

	return nil
}

// themePackageFromDir derives the theme's Go package name from its directory
// name, dropping the conventional "airway-" prefix and "-theme" suffix
// (airway-terminal-theme -> terminal) and removing remaining separators.
func themePackageFromDir(dirName string) string {
	name := strings.TrimPrefix(dirName, "airway-")
	name = strings.TrimSuffix(name, "-theme")
	name = strings.ToLower(name)
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		default:
			return -1
		}
	}, name)
}

func printCLIThemeNewUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway theme:new [--local[=path]] <module-path | path>")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "examples:")
	_, _ = fmt.Fprintln(w, "  airway theme:new sunset                                 # creates ./sunset, module sunset")
	_, _ = fmt.Fprintln(w, "  airway theme:new github.com/me/airway-sunset-theme      # package: sunset")
	_, _ = fmt.Fprintln(w, "  airway theme:new /tmp/airway-sunset-theme               # creates the theme at that path")
	_, _ = fmt.Fprintln(w, "  airway theme:new --local sunset-theme                   # develop against the airway checkout in $PWD")
}
