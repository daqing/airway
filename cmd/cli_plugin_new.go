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

var pluginNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

func runCLIPluginNew(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIPluginNewUsage(os.Stdout)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: airway plugin:new <module-path> | <path>")
	}

	return newPlugin(strings.TrimSpace(args[0]), true)
}

// newPlugin scaffolds a plugin module. ref is either a module path — the
// plugin is created under the working directory at its last segment — or a
// filesystem path (absolute, or relative like ., ./x, ../x, some/dir/x), in
// which case the plugin is created at that path and the module path is the
// path's last segment.
func newPlugin(ref string, tidy bool) error {
	module := ref
	destDir := ""
	dirName := ref

	if isDirRef(ref) || isRelativeDirPath(ref) {
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

	name := pluginNameFromDir(dirName)
	if !pluginNamePattern.MatchString(name) {
		return fmt.Errorf("cannot derive a plugin name from %q; use a module path whose last segment is the plugin name (e.g. im or github.com/me/airway-im-plugin)", ref)
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

	if err := clitemplate.ScaffoldPlugin(destDir, module, name); err != nil {
		return fmt.Errorf("scaffold plugin: %w", err)
	}

	fmt.Printf("Created a new Airway plugin in %s (module %s, name %s)\n", destDir, module, name)

	if tidy {
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = destDir
		tidyCmd.Stdout = os.Stdout
		tidyCmd.Stderr = os.Stderr
		if err := tidyCmd.Run(); err != nil {
			fmt.Printf("WARNING: `go mod tidy` failed: %v\nRun `go get github.com/daqing/airway@latest && go mod tidy` manually inside %s before building.\n", err, destDir)
		}
	}

	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s\n", destDir)
	fmt.Println("  go get github.com/daqing/airway@latest   # if tidy did not resolve it")
	fmt.Println("  go mod tidy")
	fmt.Println("\nEnable it in a host app with a blank import in plugins.go:")
	fmt.Printf("  _ \"%s\"\n", module)

	return nil
}

// isDirRef reports whether ref names a filesystem path rather than a module
// path: absolute, an explicit relative form (./ or ../), or the current
// directory itself. Module paths never begin with a slash or a dot, so the
// two forms cannot collide.
func isDirRef(ref string) bool {
	return ref == "." || ref == ".." ||
		strings.HasPrefix(ref, "/") ||
		strings.HasPrefix(ref, "./") ||
		strings.HasPrefix(ref, "../")
}

// pluginNameFromDir derives the plugin name from the directory name, dropping
// the conventional "airway-" prefix and "-plugin" suffix
// (airway-im-plugin -> im).
func pluginNameFromDir(dirName string) string {
	name := strings.TrimPrefix(dirName, "airway-")
	return strings.TrimSuffix(name, "-plugin")
}

// pluginNameFromModule derives the plugin name from a module path: the last
// path segment, minus the conventional affixes
// (github.com/me/airway-im-plugin -> im). A bare name (im) passes through
// unchanged.
func pluginNameFromModule(module string) string {
	dirName := module
	if idx := strings.LastIndex(module, "/"); idx >= 0 {
		dirName = module[idx+1:]
	}

	return pluginNameFromDir(dirName)
}

func printCLIPluginNewUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway plugin:new <module-path> | <path>")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "examples:")
	_, _ = fmt.Fprintln(w, "  airway plugin:new im                              # creates ./im, module im")
	_, _ = fmt.Fprintln(w, "  airway plugin:new github.com/me/airway-im-plugin  # creates ./airway-im-plugin")
	_, _ = fmt.Fprintln(w, "  airway plugin:new /tmp/airway-im-plugin           # creates the plugin at /tmp/airway-im-plugin;")
	_, _ = fmt.Fprintln(w, "                                                    # the module path is its last segment")
	_, _ = fmt.Fprintln(w, "  airway plugin:new sites/airway-im-plugin          # relative path: same rule")
}
