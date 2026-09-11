// Package clitemplate embeds the Airway project template used by
// `airway new` to scaffold a new application.
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

//go:embed all:enginetemplate
var engineTemplateFS embed.FS

// ModulePlaceholder is replaced with the new project's module path in every
// template file.
const ModulePlaceholder = "{{module}}"

// EnginePlaceholder is replaced with the engine name in every engine template
// file, including file and directory names.
const EnginePlaceholder = "{{engine}}"

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

// ScaffoldEngine writes the embedded engine template into destDir, replacing
// ModulePlaceholder with module and EnginePlaceholder with name in file
// contents. EnginePlaceholder is also replaced in file and directory names
// (e.g. app/api/{{engine}}_api). Template files carry a .tmpl suffix which is
// stripped on write.
func ScaffoldEngine(destDir string, module string, name string) error {
	return fs.WalkDir(engineTemplateFS, "enginetemplate", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel("enginetemplate", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(destDir, strings.ReplaceAll(rel, EnginePlaceholder, name))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		target = strings.TrimSuffix(target, ".tmpl")

		data, err := engineTemplateFS.ReadFile(path)
		if err != nil {
			return err
		}

		content := strings.ReplaceAll(string(data), ModulePlaceholder, module)
		content = strings.ReplaceAll(content, EnginePlaceholder, name)

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		return os.WriteFile(target, []byte(content), 0o644)
	})
}
