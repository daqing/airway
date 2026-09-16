package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/daqing/airway/cmd/clitemplate"
)

var modulePathPattern = regexp.MustCompile(`^[a-z0-9]+([\w./-]*[\w.])?$`)

// resolveNewTarget maps the `airway new` argument to the destination directory
// and the Go module path. An absolute filesystem path (e.g.
// /path/to/foobar) creates the project at that location, with the last path
// segment as the module path; anything else is treated as the module path
// itself, and the project is created in a subdirectory of the current
// directory named after its last segment.
func resolveNewTarget(arg string) (destDir string, module string, err error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", "", fmt.Errorf("invalid module path %q", arg)
	}

	if filepath.IsAbs(arg) {
		destDir = filepath.Clean(arg)
		module = filepath.Base(destDir)
	} else {
		module = arg
		dirName := module
		if idx := strings.LastIndex(module, "/"); idx >= 0 {
			dirName = module[idx+1:]
		}
		destDir = filepath.Join(".", dirName)
	}

	if !modulePathPattern.MatchString(module) {
		return "", "", fmt.Errorf("invalid module path %q", module)
	}

	return destDir, module, nil
}

func runCLINew(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLINewUsage(os.Stdout)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: airway new <module-path | directory>")
	}

	return newProject(strings.TrimSpace(args[0]), true)
}

func newProject(arg string, tidy bool) error {
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

	if err := clitemplate.Scaffold(destDir, module); err != nil {
		return fmt.Errorf("scaffold project: %w", err)
	}

	if err := rewriteScaffoldPort(destDir); err != nil {
		return fmt.Errorf("rewrite scaffold port: %w", err)
	}

	if err := copyScaffoldEnv(destDir); err != nil {
		return fmt.Errorf("copy .env.example: %w", err)
	}

	fmt.Printf("Created a new Airway project in %s (module %s)\n", destDir, module)

	pinned, err := pinScaffoldAirwayVersion(destDir)
	if err != nil {
		return fmt.Errorf("pin airway version in go.mod: %w", err)
	}

	if tidy {
		if pinned && !airwayVersionResolvable(destDir) {
			// The pinned version is not published yet (e.g. a locally built
			// CLI ahead of the tags). Drop the pin upfront: letting tidy
			// discover the failure itself would fall back from the proxy to
			// a direct git fetch, which can hang for minutes on a slow or
			// blocked network.
			if err := unpinScaffoldAirwayVersion(destDir); err == nil {
				pinned = false
			}
		}

		err := tidyScaffold(destDir)
		if err != nil && pinned {
			// Tidy failed with the pin in place; drop it and let tidy fall
			// back to discovering a version from the proxy.
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
	fmt.Println("  # edit .env — set AIRWAY_DB_DSN and AIRWAY_PORT")
	fmt.Println("  airway db:create")
	fmt.Println("  airway db:migrate")
	fmt.Println("  go run .               # start the HTTP server")

	return nil
}

// scaffoldCommandTimeout bounds every external command run during
// scaffolding. Network-backed commands (`go mod tidy`, `go list`) can stall
// for a long time when the module proxy falls back to direct VCS access on a
// slow or blocked network, so they must not run without a deadline.
const scaffoldCommandTimeout = 3 * time.Minute

// runScaffoldCommand runs an external command in dir with a timeout and git
// terminal prompts disabled (so git fails fast instead of blocking on a
// credential prompt). When quiet is set, the command's output is discarded.
func runScaffoldCommand(dir string, timeout time.Duration, quiet bool, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if !quiet {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("`%s %s` timed out after %s", name, strings.Join(args, " "), timeout)
		}
		return err
	}

	return nil
}

// initScaffoldGit initializes a git repository in the new project, so it is
// ready for the first commit right after scaffolding.
func initScaffoldGit(destDir string) error {
	return runScaffoldCommand(destDir, scaffoldCommandTimeout, false, "git", "init")
}

// pinScaffoldAirwayVersion writes a require directive for the framework at
// the CLI's own version into the scaffolded go.mod, so the first
// `go mod tidy` downloads a known version instead of searching the proxy for
// a module that provides each airway package (which needs direct network
// access when the module is not proxied). Dev builds without a VERSION file
// skip this step. It reports whether a pin was written.
func pinScaffoldAirwayVersion(destDir string) (bool, error) {
	if !strings.HasPrefix(Version, "v") {
		return false, nil
	}

	if err := runScaffoldCommand(destDir, scaffoldCommandTimeout, false, "go", "mod", "edit", "-require", "github.com/daqing/airway@"+Version); err != nil {
		return false, err
	}

	return true, nil
}

func unpinScaffoldAirwayVersion(destDir string) error {
	if err := runScaffoldCommand(destDir, scaffoldCommandTimeout, false, "go", "mod", "edit", "-droprequire", "github.com/daqing/airway"); err == nil {
		return nil
	}

	// `go mod edit` refuses to run when the pinned require itself makes
	// go.mod unparsable (e.g. a dev version like v9.9.9); strip the require
	// line textually instead.
	goMod := filepath.Join(destDir, "go.mod")
	data, err := os.ReadFile(goMod)
	if err != nil {
		return err
	}

	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "github.com/daqing/airway") {
			continue
		}
		kept = append(kept, line)
	}

	return os.WriteFile(goMod, []byte(strings.Join(kept, "\n")), 0o644)
}

// airwayVersionResolvable reports whether the CLI's own version can be
// resolved from the module proxy. The query is short-timeout best-effort: on
// any failure (including a stalled network) the caller drops the pin instead
// of letting `go mod tidy` hang on a direct VCS fallback.
func airwayVersionResolvable(destDir string) bool {
	return runScaffoldCommand(destDir, 30*time.Second, true, "go", "list", "-m", "github.com/daqing/airway@"+Version) == nil
}

func tidyScaffold(destDir string) error {
	return runScaffoldCommand(destDir, scaffoldCommandTimeout, false, "go", "mod", "tidy")
}

// copyScaffoldEnv seeds the new project's .env from its .env.example, so the
// app runs without a manual copy step.
func copyScaffoldEnv(destDir string) error {
	data, err := os.ReadFile(filepath.Join(destDir, ".env.example"))
	if err != nil {
		return err
	}

	dst := filepath.Join(destDir, ".env")
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.WriteFile(dst, data, 0o644)
}

// scaffoldPortFiles are the scaffolded files whose default port is rewritten
// from 1900 (the framework repo's own default) to 1905, so a freshly generated
// app doesn't collide with a locally running airway server.
var scaffoldPortFiles = []string{
	".env.example",
	"Dockerfile",
	"app/views/home/index.templ",
	"app/views/home/index_templ.go",
}

func rewriteScaffoldPort(destDir string) error {
	for _, rel := range scaffoldPortFiles {
		path := filepath.Join(destDir, rel)

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		updated := strings.ReplaceAll(string(data), "1900", "1905")
		if updated == string(data) {
			continue
		}

		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func printCLINewUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway new <module-path | directory>")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "examples:")
	_, _ = fmt.Fprintln(w, "  airway new myapp")
	_, _ = fmt.Fprintln(w, "  airway new github.com/me/myapp")
	_, _ = fmt.Fprintln(w, "  airway new /path/to/myapp    # create at that path; module: myapp")
}
