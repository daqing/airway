package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/daqing/airway/cmd/clitemplate"
)

const desktopDir = "desktop"

// desktopWailsVersion pins the Wails v3 release the generated wrapper is
// built against; desktop:init adds it to the host project's go.mod.
const desktopWailsVersion = "v3.0.0-beta.24"

func runCLIDesktopInit(args []string) error {
	force := false
	for _, arg := range args {
		if isHelpArg(arg) {
			printCLIDesktopInitUsage(os.Stdout)
			return nil
		}

		if arg == "--force" {
			force = true
			continue
		}

		return fmt.Errorf("unknown argument %q; usage: airway desktop:init [--force]", arg)
	}

	module, err := modulePathAt(".")
	if err != nil {
		return err
	}

	appName := desktopAppName(module)
	bundleID := desktopBundleID(module)
	plugins := hostPluginImports()

	mainPath := filepath.Join(desktopDir, "main.go")
	regenerate := force

	if _, err := os.Stat(mainPath); err != nil {
		regenerate = true
	}

	if regenerate {
		if err := clitemplate.ScaffoldDesktop(desktopDir, module, appName, bundleID, plugins); err != nil {
			return err
		}
		fmt.Printf("Generated desktop target in %s (app %q, bundle %q)\n", desktopDir, appName, bundleID)
	} else {
		if err := clitemplate.ScaffoldDesktopMigrations(desktopDir, module, appName, plugins); err != nil {
			return err
		}
		fmt.Printf("Refreshed %s/migrations.go (desktop/main.go left untouched; use --force to regenerate)\n", desktopDir)
	}

	copied, err := syncDesktopMigrations()
	if err != nil {
		return err
	}
	fmt.Printf("Synced %d migration file(s) into %s/migrations\n", copied, desktopDir)

	if err := addDesktopDependency(); err != nil {
		fmt.Printf("warning: could not run `go get` automatically (%v)\n", err)
		fmt.Printf("         run it manually: go get github.com/wailsapp/wails/v3@%s\n", desktopWailsVersion)
	}

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Install the desktop toolchain (once):")
	fmt.Printf("       go install github.com/wailsapp/wails/v3/cmd/wails3@%s\n", desktopWailsVersion)
	fmt.Println("       go install github.com/go-task/task/v3/cmd/task@latest")
	fmt.Println("  2. Build and package:")
	fmt.Printf("       cd %s && wails3 task build       # dev build\n", desktopDir)
	fmt.Printf("       cd %s && wails3 task package     # .app / NSIS / deb+rpm\n", desktopDir)
	fmt.Println("  3. Re-run `airway desktop:init` after adding migrations or plugins;")
	fmt.Println("     it refreshes the migration copy and the plugin mirror without")
	fmt.Println("     touching desktop/main.go.")

	return nil
}

// addDesktopDependency pins the Wails v3 module in the host project's
// go.mod, mirroring what plugin:install does for plugin modules. It is a
// variable so tests can stub it out.
var addDesktopDependency = func() error {
	dep := "github.com/wailsapp/wails/v3@" + desktopWailsVersion
	fmt.Printf("Adding %s to go.mod...\n", dep)

	cmd := exec.Command("go", "get", dep)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// syncDesktopMigrations copies the project's SQL migration pairs into the
// desktop wrapper's embedded migrations directory. Files removed from
// db/migrate stay behind on purpose: re-running desktop:init never deletes.
func syncDesktopMigrations() (int, error) {
	entries, err := os.ReadDir(filepath.Join("db", "migrate"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	target := filepath.Join(desktopDir, "migrations")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return 0, err
	}

	copied := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(name, ".up.sql") && !strings.HasSuffix(name, ".down.sql") {
			continue
		}

		data, err := os.ReadFile(filepath.Join("db", "migrate", name))
		if err != nil {
			return copied, err
		}

		if err := os.WriteFile(filepath.Join(target, name), data, 0o644); err != nil {
			return copied, err
		}
		copied++
	}

	return copied, nil
}

// desktopAppName derives the app name from the module path's last segment,
// keeping only characters that are safe in binary, bundle and directory
// names.
func desktopAppName(module string) string {
	base := module
	if idx := strings.LastIndex(module, "/"); idx >= 0 {
		base = module[idx+1:]
	}

	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		}
	}

	if b.Len() == 0 {
		return "airway-app"
	}

	return b.String()
}

// desktopBundleID derives a reverse-DNS bundle identifier from the module
// path: github.com/user/app → com.user.app; bare modules keep their segments.
func desktopBundleID(module string) string {
	parts := strings.Split(strings.Trim(module, "/"), "/")

	knownHosts := map[string]bool{
		"github.com": true, "gitlab.com": true, "gitee.com": true, "bitbucket.org": true,
	}
	if len(parts) >= 3 && knownHosts[parts[0]] {
		parts = parts[1:]
	}

	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.NewReplacer("_", "-", ".", "").Replace(part)
		if part != "" {
			segments = append(segments, part)
		}
	}

	if len(segments) == 0 {
		return "com.airway.app"
	}

	return "com." + strings.Join(segments, ".")
}

var pluginImportPattern = regexp.MustCompile(`(?m)^\s*_\s+"([^"]+)"`)

// hostPluginImports mirrors the blank imports of the project's root
// plugins.go, so the desktop binary compiles the same plugins as the web
// binary.
func hostPluginImports() []string {
	data, err := os.ReadFile("plugins.go")
	if err != nil {
		return nil
	}

	matches := pluginImportPattern.FindAllStringSubmatch(string(data), -1)
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		paths = append(paths, match[1])
	}

	return paths
}

func printCLIDesktopInitUsage(w *os.File) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway desktop:init [--force]    generate the Wails v3 desktop target in ./desktop")
}
