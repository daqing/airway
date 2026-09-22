package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/daqing/airway/cmd/clitemplate"
)

// ssgThemeModule is the showcase theme the scaffolded site depends on.
// `ssg:new --local` points it at <checkout>/themes/corporate; published
// theme versions are tagged themes/corporate/vX.Y.Z in the framework repo.
const ssgThemeModule = "github.com/daqing/airway/themes/corporate"

func runCLISSGNew(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLISSGNewUsage(os.Stdout)
		return nil
	}

	// --local[=path] points the scaffolded site at an airway source checkout
	// through replace directives — the framework and the corporate theme
	// both resolve from disk, so unreleased template APIs keep working.
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
		return fmt.Errorf("usage: airway ssg:new [--local[=path]] <module-path | directory>")
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

	return newSSGProject(strings.TrimSpace(rest[0]), true, local)
}

func newSSGProject(arg string, tidy bool, localDir string) error {
	destDir, module, err := resolveNewTarget(arg)
	if err != nil {
		return err
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

	if err := clitemplate.ScaffoldSSG(destDir, module); err != nil {
		return fmt.Errorf("scaffold site project: %w", err)
	}

	fmt.Printf("Created a new showcase site in %s (module %s, theme corporate)\n", destDir, module)

	pinned := false

	if localDir != "" {
		if err := replaceScaffoldAirway(destDir, localDir); err != nil {
			return fmt.Errorf("replace airway with local checkout: %w", err)
		}
		if err := runScaffoldCommand(destDir, scaffoldCommandTimeout, false,
			"go", "mod", "edit", "-replace", ssgThemeModule+"="+filepath.Join(localDir, "themes", "corporate")); err != nil {
			return fmt.Errorf("replace theme with local checkout: %w", err)
		}
		fmt.Printf("Replaced the framework and the corporate theme with the local checkout %s\n", localDir)
	} else {
		// Pin both modules at the CLI's own version so the first tidy pulls
		// known versions; the pins are dropped when they do not resolve (e.g.
		// a theme version that is not tagged yet), falling back to proxy
		// discovery with a manual-tidy warning.
		if pinned, err = pinScaffoldAirwayVersion(destDir); err != nil {
			return fmt.Errorf("pin airway version in go.mod: %w", err)
		}

		if strings.HasPrefix(Version, "v") {
			if err := runScaffoldCommand(destDir, scaffoldCommandTimeout, false,
				"go", "mod", "edit", "-require", ssgThemeModule+"@"+Version); err != nil {
				return fmt.Errorf("pin theme version in go.mod: %w", err)
			}
			pinned = true
		}
	}

	if tidy {
		err := tidyScaffold(destDir)
		if err != nil && pinned && localDir == "" {
			// unpinScaffoldAirwayVersion strips every go.mod line naming the
			// framework — including the theme require, whose module path
			// embeds the framework path.
			if unpinErr := unpinScaffoldAirwayVersion(destDir); unpinErr == nil {
				err = tidyScaffold(destDir)
			}
		}
		if err != nil {
			fmt.Printf("WARNING: `go mod tidy` failed: %v\nRun it manually inside %s before building.\n", err, destDir)
		}
	}

	if err := initScaffoldGit(destDir); err != nil {
		fmt.Printf("WARNING: `git init` failed: %v\n", err)
	}

	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", destDir)
	fmt.Println("  # edit ssg.go — site metadata, pages, and theme data")
	fmt.Println("  airway ssg:build           # export the static site into dist/")
	fmt.Println("  airway ssg:serve           # preview on http://127.0.0.1:3000")

	return nil
}

func printCLISSGNewUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway ssg:new [--local[=path]] <module-path | directory>")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "examples:")
	_, _ = fmt.Fprintln(w, "  airway ssg:new mysite")
	_, _ = fmt.Fprintln(w, "  airway ssg:new /path/to/mysite   # create at that path; module: mysite")
	_, _ = fmt.Fprintln(w, "  airway ssg:new --local mysite    # use the airway checkout in $PWD")
}
