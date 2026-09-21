// Package clitemplate embeds the Airway project template used by
// `airway new` to scaffold a new application, and the desktop wrapper
// template used by `airway desktop:init`.
package clitemplate

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:template
var templateFS embed.FS

//go:embed all:plugintemplate
var pluginTemplateFS embed.FS

//go:embed all:desktop
var desktopTemplateFS embed.FS

// ModulePlaceholder is replaced with the new project's module path in every
// template file.
const ModulePlaceholder = "{{module}}"

// PluginPlaceholder is replaced with the plugin name in every plugin template
// file, including file and directory names.
const PluginPlaceholder = "{{plugin}}"

// AppNamePlaceholder is replaced with the desktop app name (derived from the
// module path) in every desktop template file.
const AppNamePlaceholder = "{{app_name}}"

// BundleIDPlaceholder is replaced with the derived bundle identifier in every
// desktop template file.
const BundleIDPlaceholder = "{{bundle_id}}"

// PluginsImportsPlaceholder is replaced with the blank-import lines mirroring
// the host project's plugins.go inside the desktop wrapper.
const PluginsImportsPlaceholder = "{{plugins_imports}}"

// PluginsEnabledPlaceholder is replaced with "true"/"false" depending on
// whether the host project blank-imports any plugin.
const PluginsEnabledPlaceholder = "{{plugins_enabled}}"

// Scaffold writes the embedded project template into destDir, replacing
// ModulePlaceholder with module. Template files carry a .tmpl suffix which is
// stripped on write.
func Scaffold(destDir string, module string) error {
	return fs.WalkDir(templateFS, "template", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel("template", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(destDir, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		target = strings.TrimSuffix(target, ".tmpl")

		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}

		content := strings.ReplaceAll(string(data), ModulePlaceholder, module)

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		return os.WriteFile(target, []byte(content), 0o644)
	})
}

// ScaffoldDesktop writes the embedded desktop wrapper template into destDir
// (usually "desktop" inside a host project), replacing ModulePlaceholder,
// AppNamePlaceholder, BundleIDPlaceholder and PluginsImportsPlaceholder.
// Template files carry a .tmpl suffix which is stripped on write. Existing
// files are overwritten in place; nothing is deleted.
func ScaffoldDesktop(destDir, module, appName, bundleID string, pluginImports []string) error {
	return scaffoldTree(desktopTemplateFS, "desktop", destDir, func(content string) string {
		return renderDesktopContent(content, module, appName, bundleID, pluginImports)
	})
}

// ScaffoldDesktopMigrations regenerates only desktop/migrations.go (the
// embed declaration plus the plugin-import mirror). desktop:init uses it on
// re-runs so user edits to main.go survive while the plugin mirror stays in
// sync.
func ScaffoldDesktopMigrations(destDir, module, appName string, pluginImports []string) error {
	content, err := renderDesktopFile("migrations.go.tmpl", module, appName, "", pluginImports)
	if err != nil {
		return err
	}

	target := filepath.Join(destDir, "migrations.go")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	return os.WriteFile(target, []byte(content), 0o644)
}

func renderDesktopFile(name, module, appName, bundleID string, pluginImports []string) (string, error) {
	data, err := desktopTemplateFS.ReadFile("desktop/" + name)
	if err != nil {
		return "", err
	}

	return renderDesktopContent(string(data), module, appName, bundleID, pluginImports), nil
}

func renderDesktopContent(content, module, appName, bundleID string, pluginImports []string) string {
	content = strings.ReplaceAll(content, ModulePlaceholder, module)
	content = strings.ReplaceAll(content, AppNamePlaceholder, appName)
	if bundleID != "" {
		content = strings.ReplaceAll(content, BundleIDPlaceholder, bundleID)
	}

	imports := ""
	for _, path := range pluginImports {
		imports += "\t_ \"" + path + "\"\n"
	}
	content = strings.ReplaceAll(content, PluginsImportsPlaceholder, imports)

	enabled := "false"
	if len(pluginImports) > 0 {
		enabled = "true"
	}
	content = strings.ReplaceAll(content, PluginsEnabledPlaceholder, enabled)

	return content
}

func scaffoldTree(fsys embed.FS, root, destDir string, transform func(string) string) error {
	return fs.WalkDir(fsys, root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(destDir, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		target = strings.TrimSuffix(target, ".tmpl")

		data, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		return os.WriteFile(target, []byte(transform(string(data))), 0o644)
	})
}

// ScaffoldPlugin writes the embedded plugin template into destDir, replacing
// ModulePlaceholder with module and PluginPlaceholder with name in file
// contents. PluginPlaceholder is also replaced in file and directory names
// (e.g. app/api/{{plugin}}_api). Template files carry a .tmpl suffix which is
// stripped on write.
func ScaffoldPlugin(destDir string, module string, name string) error {
	return fs.WalkDir(pluginTemplateFS, "plugintemplate", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel("plugintemplate", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(destDir, strings.ReplaceAll(rel, PluginPlaceholder, name))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		target = strings.TrimSuffix(target, ".tmpl")

		data, err := pluginTemplateFS.ReadFile(path)
		if err != nil {
			return err
		}

		content := strings.ReplaceAll(string(data), ModulePlaceholder, module)
		content = strings.ReplaceAll(content, PluginPlaceholder, name)

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		return os.WriteFile(target, []byte(content), 0o644)
	})
}
