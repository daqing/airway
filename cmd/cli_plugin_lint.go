package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// pluginLegacyDirs are the top-level directories of the pre-install/ plugin
// layout. plugin:install reads only install/, so plugin:lint asks for them
// to be moved under it.
var pluginLegacyDirs = []string{"app", "host", "deps", "ignore"}

// pluginLegacyInstallDirs maps transitional subdirectory names inside
// install/ to their current ones. install/app held the plugin's compiled-in
// implementation before the directory was named install/lib/.
var pluginLegacyInstallDirs = map[string]string{
	"app": pluginLibDir,
}

func runCLIPluginLint(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIPluginLintUsage(os.Stdout)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: airway plugin:lint")
	}

	if !pluginProjectAt(".") {
		return fmt.Errorf("the current directory is not an Airway plugin project (a go.mod requiring github.com/daqing/airway, without a main.go)")
	}

	issues := 0
	color := stdoutIsTerminal()
	for _, dir := range pluginLegacyDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			issues++
			fmt.Printf("%s\n", lintFindingMessage(dir, color))
		}
	}
	for _, from := range sortedLegacyInstallDirs() {
		legacy := filepath.Join(pluginInstallDir, from)
		if info, err := os.Stat(legacy); err == nil && info.IsDir() {
			issues++
			fmt.Printf("%s\n", lintInstallFindingMessage(legacy, pluginLegacyInstallDirs[from], color))
		}
	}

	if issues == 0 {
		fmt.Println("no issues found")
		return nil
	}

	return fmt.Errorf("plugin lint found %d issue(s)", issues)
}

// sortedLegacyInstallDirs returns the keys of pluginLegacyInstallDirs in
// sorted order, so findings print deterministically.
func sortedLegacyInstallDirs() []string {
	names := make([]string, 0, len(pluginLegacyInstallDirs))
	for name := range pluginLegacyInstallDirs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// lintFindingMessage renders one finding: the label before the colon in
// light blue, the hint after it in dark yellow.
func lintFindingMessage(dir string, color bool) string {
	return highlightLightBlue(fmt.Sprintf("legacy top-level %s/", dir), color) +
		": " +
		highlightDarkYellow(fmt.Sprintf("plugin:install no longer reads it; move it to install/%s", dir), color)
}

// lintInstallFindingMessage renders a finding for a legacy install/
// subdirectory that has been renamed.
func lintInstallFindingMessage(legacy, current string, color bool) string {
	return highlightLightBlue(fmt.Sprintf("legacy %s/", legacy), color) +
		": " +
		highlightDarkYellow(fmt.Sprintf("move it to install/%s", current), color)
}

// highlightLightBlue wraps s in light-blue ANSI codes when enabled; colors
// are emitted only when stdout is a terminal, so piped output stays plain.
func highlightLightBlue(s string, enabled bool) string {
	return highlightANSI(s, enabled, "94")
}

// highlightDarkYellow wraps s in dark-yellow ANSI codes when enabled.
func highlightDarkYellow(s string, enabled bool) string {
	return highlightANSI(s, enabled, "33")
}

func highlightANSI(s string, enabled bool, code string) string {
	if !enabled {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func stdoutIsTerminal() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// pluginProjectAt reports whether dir holds an Airway plugin project: a go.mod
// requiring github.com/daqing/airway with a module path of its own and no
// main.go beside it — host applications ship one, and the framework
// repository is excluded by its module path.
func pluginProjectAt(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "main.go")); err == nil {
		return false
	}

	module, err := modulePathAt(dir)
	if err != nil || module == airwayModulePath {
		return false
	}

	_, ok := goModAirwayVersion(dir)
	return ok
}

func printCLIPluginLintUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway plugin:lint")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "checks the current Airway plugin project for layout issues,")
	_, _ = fmt.Fprintln(w, "such as legacy top-level directories that plugin:install no longer reads,")
	_, _ = fmt.Fprintln(w, "or install/ subdirectories that have been renamed")
}
