package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPluginScaffoldsModule(t *testing.T) {
	wd := useTempWorkingDir(t)

	if err := newPlugin("im", false); err != nil {
		t.Fatalf("new plugin: %v", err)
	}

	pluginFile := readFile(t, filepath.Join(wd, "im", "plugin.go"))
	if !strings.Contains(pluginFile, "package implugin") {
		t.Fatalf("expected implugin package, got:\n%s", pluginFile)
	}
	if !strings.Contains(pluginFile, `func (Plugin) Name() string      { return "im" }`) {
		t.Fatalf("expected plugin name in plugin.go, got:\n%s", pluginFile)
	}
	if !strings.Contains(pluginFile, `"im/install/lib/api/im_api"`) {
		t.Fatalf("expected plugin api import, got:\n%s", pluginFile)
	}

	goMod := readFile(t, filepath.Join(wd, "im", "go.mod"))
	if !strings.Contains(goMod, "module im") {
		t.Fatalf("expected module path in go.mod, got:\n%s", goMod)
	}

	routes := readFile(t, filepath.Join(wd, "im", "install", "lib", "api", "im_api", "routes.go"))
	if !strings.Contains(routes, "package im_api") {
		t.Fatalf("expected im_api package, got:\n%s", routes)
	}

	// The scaffolded install/ignore/ directory holds local-only files that
	// plugin:install never ships to a host project.
	if info, err := os.Stat(filepath.Join(wd, "im", "install", "ignore", ".keep")); err != nil || info.IsDir() {
		t.Fatalf("expected the scaffolded install/ignore/.keep: %v", err)
	}
}

func TestNewPluginDerivesNameFromModulePath(t *testing.T) {
	wd := useTempWorkingDir(t)

	if err := newPlugin("github.com/example/airway-billing-plugin", false); err != nil {
		t.Fatalf("new plugin: %v", err)
	}

	pluginFile := readFile(t, filepath.Join(wd, "airway-billing-plugin", "plugin.go"))
	if !strings.Contains(pluginFile, `{ return "billing" }`) {
		t.Fatalf("expected derived plugin name, got:\n%s", pluginFile)
	}
	if !strings.Contains(pluginFile, `"github.com/example/airway-billing-plugin/install/lib/api/billing_api"`) {
		t.Fatalf("expected module import rewritten, got:\n%s", pluginFile)
	}
}

func TestPluginNameFromModule(t *testing.T) {
	cases := map[string]string{
		"github.com/daqing/airway-im-plugin": "im",
		"github.com/me/billing":              "billing",
		"airway-im-plugin":                   "im",
		"im":                                 "im",
	}

	for module, want := range cases {
		if got := pluginNameFromModule(module); got != want {
			t.Fatalf("pluginNameFromModule(%q) = %q, want %q", module, got, want)
		}
	}
}

func TestNewPluginRejectsExistingNonEmptyDirectory(t *testing.T) {
	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "im"))
	writeFile(t, filepath.Join(wd, "im", "existing.txt"), "occupied\n")

	if err := newPlugin("im", false); err == nil {
		t.Fatalf("expected error for non-empty directory")
	}

	// The same check applies to the path form.
	dest := filepath.Join(wd, "airway-occupied-plugin")
	makeDirs(t, dest)
	writeFile(t, filepath.Join(dest, "existing.txt"), "occupied\n")
	if err := newPlugin(dest, false); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("expected a not-empty error for the path form, got: %v", err)
	}
}

func TestNewPluginRejectsInvalidModulePath(t *testing.T) {
	useTempWorkingDir(t)

	for _, module := range []string{"", "/leading-slash", "UPPER Case", "has space"} {
		if err := newPlugin(module, false); err == nil {
			t.Fatalf("expected error for module path %q", module)
		}
	}
}

func TestRunPluginNewHelpPrintsUsage(t *testing.T) {
	output := captureStdout(t, func() {
		if err := run([]string{"plugin:new", "-h"}); err != nil {
			t.Fatalf("run plugin:new help: %v", err)
		}
	})

	if !strings.Contains(output, "airway plugin:new <module-path> | <path>") {
		t.Fatalf("expected plugin:new usage output, got:\n%s", output)
	}
}

func TestNewPluginScaffoldsAtPath(t *testing.T) {
	wd := useTempWorkingDir(t)

	dest := filepath.Join(wd, "sites", "airway-foo-plugin")
	if err := newPlugin(dest, false); err != nil {
		t.Fatalf("new plugin: %v", err)
	}

	pluginFile := readFile(t, filepath.Join(dest, "plugin.go"))
	if !strings.Contains(pluginFile, "package fooplugin") {
		t.Fatalf("expected fooplugin package, got:\n%s", pluginFile)
	}
	if !strings.Contains(pluginFile, `{ return "foo" }`) {
		t.Fatalf("expected derived plugin name, got:\n%s", pluginFile)
	}
	if !strings.Contains(pluginFile, `"airway-foo-plugin/install/lib/api/foo_api"`) {
		t.Fatalf("expected plugin api import, got:\n%s", pluginFile)
	}

	goMod := readFile(t, filepath.Join(dest, "go.mod"))
	if !strings.Contains(goMod, "module airway-foo-plugin") {
		t.Fatalf("expected module airway-foo-plugin in go.mod, got:\n%s", goMod)
	}
}

func TestNewPluginScaffoldsAtRelativePath(t *testing.T) {
	wd := useTempWorkingDir(t)

	if err := newPlugin("./bar", false); err != nil {
		t.Fatalf("new plugin: %v", err)
	}

	pluginFile := readFile(t, filepath.Join(wd, "bar", "plugin.go"))
	if !strings.Contains(pluginFile, "package barplugin") {
		t.Fatalf("expected barplugin package, got:\n%s", pluginFile)
	}
}

func TestNewPluginTreatsDotlessRelativePathAsDirectory(t *testing.T) {
	wd := useTempWorkingDir(t)

	// A dotless multi-element path is not a valid Go module path; it must
	// scaffold in place with the last segment as the module instead.
	if err := newPlugin("sites/airway-baz-plugin", false); err != nil {
		t.Fatalf("new plugin: %v", err)
	}

	goMod := readFile(t, filepath.Join(wd, "sites", "airway-baz-plugin", "go.mod"))
	if !strings.Contains(goMod, "module airway-baz-plugin") {
		t.Fatalf("expected module airway-baz-plugin in go.mod, got:\n%s", goMod)
	}
}
