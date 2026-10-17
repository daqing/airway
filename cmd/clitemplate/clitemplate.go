// Package clitemplate embeds the Airway project template used by
// `airway new` to scaffold a new application, the desktop wrapper template
// used by `airway desktop:init`, the static showcase site template used by
// `airway ssg:new`, and the site theme template used by `airway theme:new`.
package clitemplate

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed all:template
var templateFS embed.FS

//go:embed all:plugintemplate
var pluginTemplateFS embed.FS

//go:embed all:desktop
var desktopTemplateFS embed.FS

//go:embed all:ssgtemplate
var ssgTemplateFS embed.FS

//go:embed all:themetemplate
var themeTemplateFS embed.FS

// ModulePlaceholder is replaced with the new project's module path in every
// template file.
const ModulePlaceholder = "{{module}}"

// PluginPlaceholder is replaced with the plugin name in every plugin template
// file, including file and directory names.
const PluginPlaceholder = "{{plugin}}"

// AppNamePlaceholder is replaced with the desktop app name (derived from the
// module path) in every desktop template file.
const AppNamePlaceholder = "{{app_name}}"

// PackagePlaceholder is replaced with the theme's Go package name in every
// theme template file.
const PackagePlaceholder = "{{package}}"

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
	files, dirs, err := renderTree(templateFS, "template", func(content string) string {
		return strings.ReplaceAll(content, ModulePlaceholder, module)
	})
	if err != nil {
		return err
	}

	for _, rel := range dirs {
		if err := os.MkdirAll(filepath.Join(destDir, filepath.FromSlash(rel)), 0o755); err != nil {
			return err
		}
	}

	paths := make([]string, 0, len(files))
	for rel := range files {
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	for _, rel := range paths {
		target := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, files[rel], 0o644); err != nil {
			return err
		}
	}

	return nil
}

// Render returns the embedded project template rendered for module, keyed by
// the path a scaffolded project uses (placeholders replaced, .tmpl suffixes
// stripped). `airway upgrade` uses it so both commands always agree on the
// skeleton they read and write.
func Render(module string) (map[string][]byte, error) {
	files, _, err := renderTree(templateFS, "template", func(content string) string {
		return strings.ReplaceAll(content, ModulePlaceholder, module)
	})
	return files, err
}

// RenderFS renders a project template tree rooted at root inside fsys. It
// serves `airway upgrade` when the baseline template comes from an older
// Airway module in the module cache rather than this binary.
func RenderFS(fsys fs.FS, root, module string) (files map[string][]byte, err error) {
	files, _, err = renderTree(fsys, root, func(content string) string {
		return strings.ReplaceAll(content, ModulePlaceholder, module)
	})
	return files, err
}

// TemplatePaths lists every scaffolded path (placeholders ignored, .tmpl
// suffixes stripped) in the embedded project template. Tests use it to keep
// the upgrade's per-file rules in sync with the template.
func TemplatePaths() ([]string, error) {
	files, err := Render("")
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(files))
	for rel := range files {
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	return paths, nil
}

// renderTree reads every file below root, applying transform to its content
// and stripping a .tmpl suffix from its path. Paths are slash-separated and
// relative to root.
func renderTree(fsys fs.FS, root string, transform func(string) string) (files map[string][]byte, dirs []string, err error) {
	files = map[string][]byte{}
	prefix := strings.TrimSuffix(root, "/") + "/"
	if root == "." {
		prefix = ""
	}

	err = fs.WalkDir(fsys, root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}

		rel := strings.TrimPrefix(path, prefix)
		if entry.IsDir() {
			dirs = append(dirs, rel)
			return nil
		}

		rel = strings.TrimSuffix(rel, ".tmpl")
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			return readErr
		}
		files[rel] = []byte(transform(string(data)))
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	sort.Strings(dirs)
	return files, dirs, nil
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

// ScaffoldSSG writes the embedded static-site project template into
// destDir, replacing ModulePlaceholder with module. Template files carry a
// .tmpl suffix which is stripped on write.
func ScaffoldSSG(destDir string, module string) error {
	return scaffoldTree(ssgTemplateFS, "ssgtemplate", destDir, func(content string) string {
		return strings.ReplaceAll(content, ModulePlaceholder, module)
	})
}

// ScaffoldTheme writes the embedded site-theme module template into
// destDir, replacing ModulePlaceholder with module and PackagePlaceholder
// with the theme's Go package name. Template files carry a .tmpl suffix
// which is stripped on write.
func ScaffoldTheme(destDir string, module string, pkg string) error {
	return scaffoldTree(themeTemplateFS, "themetemplate", destDir, func(content string) string {
		content = strings.ReplaceAll(content, ModulePlaceholder, module)
		return strings.ReplaceAll(content, PackagePlaceholder, pkg)
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
