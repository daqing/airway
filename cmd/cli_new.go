package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/daqing/airway/cmd/clitemplate"
)

var modulePathPattern = regexp.MustCompile(`^[a-z0-9]+([\w./-]*[\w.])?$`)

func runCLINew(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLINewUsage(os.Stdout)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: airway new <module-path>")
	}

	return newProject(strings.TrimSpace(args[0]), true)
}

func newProject(module string, tidy bool) error {
	if !modulePathPattern.MatchString(module) {
		return fmt.Errorf("invalid module path %q", module)
	}

	dirName := module
	if idx := strings.LastIndex(module, "/"); idx >= 0 {
		dirName = module[idx+1:]
	}

	destDir := filepath.Join(".", dirName)
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

	if tidy {
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = destDir
		tidyCmd.Stdout = os.Stdout
		tidyCmd.Stderr = os.Stderr
		if err := tidyCmd.Run(); err != nil {
			fmt.Printf("WARNING: `go mod tidy` failed: %v\nRun it manually inside %s before building.\n", err, destDir)
		}
	}

	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", destDir)
	fmt.Println("  # edit .env — set AIRWAY_DB_DSN and AIRWAY_PORT")
	fmt.Println("  airway db:create")
	fmt.Println("  airway db:migrate")
	fmt.Println("  go run .               # start the HTTP server")

	return nil
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
	_, _ = fmt.Fprintln(w, "  airway new <module-path>")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "examples:")
	_, _ = fmt.Fprintln(w, "  airway new myapp")
	_, _ = fmt.Fprintln(w, "  airway new github.com/me/myapp")
}
