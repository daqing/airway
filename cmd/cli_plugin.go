package cmd

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/daqing/airway/lib/plugin"
)

func runCLIPlugin(args []string) error {
	printCLIPluginUsage(os.Stdout)
	return nil
}

func printCLIPluginUsage(w *os.File) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway plugin:new <module-path>")
	_, _ = fmt.Fprintln(w, "  airway plugin:list")
	_, _ = fmt.Fprintln(w, "  airway plugin:install <module>[@version] | <local-dir>")
	_, _ = fmt.Fprintln(w, "  airway plugin:lint")
}

func runCLIPluginList() error {
	plugins := plugin.Plugins()
	if len(plugins) == 0 {
		fmt.Println("No plugins registered (add blank imports to plugins.go)")
		return nil
	}

	for _, p := range plugins {
		fmt.Printf("%s\t%s\n", p.Name(), p.MountPath())
	}

	return nil
}

// runCLIPluginInstall copies a plugin's SQL migrations (from its
// install/host/db/migrate directory) into the host's db/migrate directory
// with fresh timestamps, so they run through the regular db:migrate /
// db:rollback / db:status machinery, mirrors the rest of the plugin's
// install/host/ tree into the host project root, and merges the plugin's
// install/deps/ directory into the host project's deps/ directory. The
// installer's view of the plugin is exactly its install/ directory: host/ and
// deps/ inside it are installed, install/ignore/ (and any ignore/ directory
// inside the installable trees) is skipped, and every other directory at the
// plugin's top level is ignored. When the plugin is read from disk, its root
// .gitignore rules exclude matching files too (node_modules, build outputs,
// ...) — a module download from the proxy only carries committed files anyway.
// The argument is the plugin's module path (e.g. github.com/daqing/airway-im-plugin),
// optionally with an @version suffix like `go get` accepts, or a local
// directory holding the plugin's source (its go.mod supplies the module path,
// wired in through a replace directive — no download needed). The plugin name
// is derived from the module's last segment, same as `plugin:new`.
//
// When the plugin is not compiled into the current binary, it is enabled
// first (blank import + go get / replace), and the install then continues in
// this same process, reading install/host/db/migrate and install/deps from the
// plugin module's on-disk directory — so the current CLI's installer logic is
// always the one used, never a possibly stale project binary.
func runCLIPluginInstall(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: airway plugin:install <module>[@version] | <local-dir>")
	}

	ref := strings.TrimSpace(args[0])

	module := ref
	local := false
	if info, err := os.Stat(ref); err == nil && info.IsDir() {
		m, err := modulePathAt(ref)
		if err != nil {
			return fmt.Errorf("read plugin module path from %s: %w", ref, err)
		}
		module, local = m, true
	} else {
		module, _, _ = strings.Cut(ref, "@")
	}

	name := pluginNameFromModule(module)
	if !pluginNamePattern.MatchString(name) {
		return fmt.Errorf("cannot derive a plugin name from %q; use the plugin's module path (e.g. github.com/daqing/airway-im-plugin)", module)
	}

	p := plugin.Find(name)
	if p == nil {
		if err := enablePlugin(name, module, ref, local); err != nil {
			return err
		}
	}

	if provider, ok := p.(plugin.MigrationProvider); ok {
		if err := installPluginMigrations(name, provider.MigrationFS(), migrationDir, timeNow(), nil); err != nil {
			return err
		}
	} else if err := installPluginMigrationsFromModule(name, module); err != nil {
		return err
	}

	if err := installPluginHost(name, module); err != nil {
		return err
	}

	return installPluginDeps(name, module)
}

// enablePlugin makes plugin:install self-contained: when the plugin is not
// compiled into the current binary, it adds the blank import to the host's
// plugins.go and wires the module into go.mod — `go mod edit -replace` +
// `go mod tidy` for a local directory (no download), otherwise `go get ref`
// (ref may carry an @version suffix).
func enablePlugin(name, module, ref string, local bool) error {
	for _, file := range []string{"go.mod", "plugins.go"} {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("plugin %q (%s) is not registered; run from the host project root (go.mod and plugins.go are required), or add its blank import to plugins.go, run `go get %s`, and retry", name, module, ref)
		}
	}

	added, err := ensurePluginImport("plugins.go", module)
	if err != nil {
		return err
	}
	if added {
		fmt.Printf("Added _ %s to plugins.go\n", strconv.Quote(module))
	}

	if local {
		if err := replacePluginWithLocalDir(module, ref); err != nil {
			return err
		}
	} else {
		fmt.Printf("Fetching %s\n", ref)
		get := exec.Command("go", "get", ref)
		get.Stdout = os.Stdout
		get.Stderr = os.Stderr
		if err := get.Run(); err != nil {
			return fmt.Errorf("go get %s: %w", ref, err)
		}
	}

	return nil
}

// replacePluginWithLocalDir points the module requirement at a local checkout
// via a replace directive, then tidies so go.mod picks up the require entry
// without contacting the network for this module.
func replacePluginWithLocalDir(module, dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	fmt.Printf("Replacing %s with local checkout %s\n", module, abs)
	edit := exec.Command("go", "mod", "edit", "-replace", module+"="+abs)
	edit.Stdout = os.Stdout
	edit.Stderr = os.Stderr
	if err := edit.Run(); err != nil {
		return fmt.Errorf("go mod edit -replace %s=%s: %w", module, abs, err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	return nil
}

// ensurePluginImport adds a blank import for module to the host's plugins.go,
// reusing an existing import block when present. It reports whether the file
// changed; an already-present import is a no-op. The file is parsed rather
// than string-matched, because the scaffolded plugins.go shows a sample
// import block inside a comment.
func ensurePluginImport(path string, module string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, parser.ImportsOnly)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", path, err)
	}

	quoted := strconv.Quote(module)
	for _, imp := range file.Imports {
		if imp.Path.Value == quoted {
			return false, nil
		}
	}

	var edited string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT || !gen.Lparen.IsValid() {
			continue
		}

		at := fset.Position(gen.Lparen).Offset + 1
		edited = string(data[:at]) + "\n\t_ " + quoted + string(data[at:])
		break
	}
	if edited == "" {
		edited = strings.TrimRight(string(data), "\n") + "\n\nimport (\n\t_ " + quoted + "\n)\n"
	}

	formatted, err := format.Source([]byte(edited))
	if err != nil {
		return false, fmt.Errorf("format %s after adding plugin import: %w", path, err)
	}

	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return false, err
	}

	return true, nil
}

// installPluginMigrations copies every up/down migration pair in the FS into
// dstDir with fresh timestamps; match, when non-nil, filters FS paths the
// plugin's .gitignore excludes.
func installPluginMigrations(pluginName string, migrations fs.FS, dstDir string, now time.Time, match func(string, bool) bool) error {
	pairs, err := collectPluginMigrationPairs(migrations, match)
	if err != nil {
		return fmt.Errorf("read plugin %s migrations: %w", pluginName, err)
	}

	if len(pairs) == 0 {
		if hasGoCodeMigrations(migrations, match) {
			fmt.Printf("Plugin %s defines its migrations in Go code; they take effect on import and need no install\n", pluginName)
		} else {
			fmt.Printf("Plugin %s has no SQL migrations to install\n", pluginName)
		}
		return nil
	}

	if err := ensureDir(dstDir); err != nil {
		return err
	}

	for i, pair := range pairs {
		version := now.Add(time.Duration(i) * time.Second).Format("20060102150405")

		if pluginMigrationInstalled(dstDir, pair.name) {
			fmt.Printf("Migration %s already installed, skipping...\n", pair.name)
			continue
		}

		for _, file := range []struct {
			content []byte
			suffix  string
		}{
			{pair.up, ".up.sql"},
			{pair.down, ".down.sql"},
		} {
			dstPath := filepath.Join(dstDir, version+"_"+pair.name+file.suffix)
			if err := os.WriteFile(dstPath, file.content, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", dstPath, err)
			}
		}

		fmt.Printf("Installed migration %s_%s from plugin %s\n", version, pair.name, pluginName)
	}

	return nil
}

type pluginMigrationPair struct {
	name string
	up   []byte
	down []byte
}

func collectPluginMigrationPairs(migrations fs.FS, match func(string, bool) bool) ([]pluginMigrationPair, error) {
	ups := map[string][]byte{}
	downs := map[string][]byte{}

	err := fs.WalkDir(migrations, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == pluginIgnoreDir {
				return fs.SkipDir
			}
			if match != nil && path != "." && match(path, true) {
				return fs.SkipDir
			}
			return nil
		}
		if match != nil && match(path, false) {
			return nil
		}

		base := filepath.Base(path)

		var suffix string
		var target map[string][]byte

		switch {
		case strings.HasSuffix(base, ".up.sql"):
			suffix = ".up.sql"
			target = ups
		case strings.HasSuffix(base, ".down.sql"):
			suffix = ".down.sql"
			target = downs
		default:
			return nil
		}

		content, err := fs.ReadFile(migrations, path)
		if err != nil {
			return err
		}

		name := strings.TrimSuffix(base, suffix)
		// Strip the plugin's own version prefix; the install assigns a fresh one.
		if _, rest, found := strings.Cut(name, "_"); found {
			name = rest
		}

		target[name] = content
		return nil
	})
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(ups))
	for name := range ups {
		if _, ok := downs[name]; !ok {
			return nil, fmt.Errorf("missing down migration for %s", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)

	pairs := make([]pluginMigrationPair, 0, len(names))
	for _, name := range names {
		pairs = append(pairs, pluginMigrationPair{name: name, up: ups[name], down: downs[name]})
	}

	return pairs, nil
}

// hasGoCodeMigrations reports whether migrations holds Go files besides
// tests — migrations written in Go code take effect on import and need no
// install, so the installer only mentions them instead of copying anything.
func hasGoCodeMigrations(migrations fs.FS, match func(string, bool) bool) bool {
	found := false

	_ = fs.WalkDir(migrations, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == pluginIgnoreDir {
				return fs.SkipDir
			}
			if match != nil && path != "." && match(path, true) {
				return fs.SkipDir
			}
			return nil
		}
		if match != nil && match(path, false) {
			return nil
		}

		base := entry.Name()
		if strings.HasSuffix(base, ".go") && !strings.HasSuffix(base, "_test.go") {
			found = true
		}
		return nil
	})

	return found
}

// pluginMigrationInstalled reports whether db/migrate already contains a
// migration whose name part matches (regardless of its timestamp prefix).
func pluginMigrationInstalled(dstDir string, name string) bool {
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		base := strings.TrimSuffix(entry.Name(), ".up.sql")
		if _, rest, found := strings.Cut(base, "_"); found && rest == name {
			return true
		}
	}

	return false
}

// pluginInstallDir is the directory inside a plugin module holding everything
// `plugin:install` reads: host/ (mirrored into the host project's own tree),
// deps/ (merged into the host project's deps/ directory), and ignore/
// (local-only files, never read). Nothing outside install/ is installed.
const pluginInstallDir = "install"

// pluginHostDir is the directory inside a plugin's install/ directory holding
// files that `plugin:install` installs into the host project's own tree,
// mirroring the host layout (install/host/docker-compose.yml → the host's
// docker-compose.yml). It is the counterpart of deps/, whose contents merge
// verbatim into the host's deps/ directory instead of joining the host
// sources. The db/migrate subtree is exempt from this verbatim mirroring:
// SQL migrations are installed with fresh timestamps by the migration
// installer, and Go DSL migrations are compiled into the plugin.
const pluginHostDir = "host"

// pluginIgnoreDir is the reserved directory name marking local-only files
// that stay in the plugin checkout and never reach the host project.
// install/ignore/ is simply never read (only install/host/ and install/deps/
// are), and inside the installable trees the installer walks — host/, deps/,
// and the migrations — everything under an ignore/ directory is skipped
// wholesale.
const pluginIgnoreDir = "ignore"

// installPluginMigrationsFromModule installs migrations for a plugin that is
// not compiled into the current binary: it reads the plugin's
// install/host/db/migrate directory from its on-disk module directory (the
// same files a MigrationProvider would embed), so the install completes in
// this process instead of re-executing through a possibly stale project
// binary.
func installPluginMigrationsFromModule(name, module string) error {
	dir, err := pluginModuleDir(module)
	if err != nil {
		return fmt.Errorf("locate plugin module %s: %w", module, err)
	}

	return installPluginMigrationsFromDir(name, dir, migrationDir)
}

// installPluginMigrationsFromDir installs SQL migrations from a plugin module
// directory on disk, reading its install/host/db/migrate subtree. The
// plugin's root .gitignore rules exclude matching paths.
func installPluginMigrationsFromDir(name, moduleDir, dstDir string) error {
	migrateDir := filepath.Join(moduleDir, pluginInstallDir, pluginHostDir, "db", "migrate")
	if info, err := os.Stat(migrateDir); err != nil || !info.IsDir() {
		fmt.Printf("Plugin %s has no SQL migrations to install\n", name)
		return nil
	}

	ignore, err := loadPluginGitIgnore(moduleDir)
	if err != nil {
		return err
	}
	var match func(string, bool) bool
	if ignore != nil {
		prefix := pluginInstallDir + "/" + pluginHostDir + "/db/migrate/"
		match = func(path string, isDir bool) bool {
			return ignore.Match(prefix+path, isDir)
		}
	}

	return installPluginMigrations(name, os.DirFS(migrateDir), dstDir, timeNow(), match)
}

// installPluginHost mirrors the plugin's install/host/ tree into the host
// project root. The plugin module directory is resolved through the host's
// module graph, so proxy downloads and local replace directives behave the same.
func installPluginHost(name, module string) error {
	dir, err := pluginModuleDir(module)
	if err != nil {
		return fmt.Errorf("locate plugin module %s: %w", module, err)
	}

	return installPluginHostFrom(name, dir, ".")
}

// installPluginHostFrom copies every file under the plugin's install/host/
// directory into dstRoot (the host project root), preserving relative paths
// and stripping a single .templ suffix, like the deps installer. The
// install/host/db/migrate subtree is excluded: SQL migrations get fresh
// timestamps from the migration installer, and Go DSL migrations are compiled
// into the plugin — neither belongs verbatim in the host tree. Any ignore/
// directory under install/host/ is skipped wholesale (local-only files never
// reach the host), and the plugin's root .gitignore rules exclude matching
// paths. Existing destination files are left untouched, mirroring
// the deps installer's skip behavior. A plugin without an install/host/
// directory installs nothing.
func installPluginHostFrom(name, moduleDir, dstRoot string) error {
	srcDir := filepath.Join(moduleDir, pluginInstallDir, pluginHostDir)
	if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
		return nil
	}

	ignore, err := loadPluginGitIgnore(moduleDir)
	if err != nil {
		return err
	}
	prefix := pluginInstallDir + "/" + pluginHostDir + "/"

	migrateDir := filepath.Join(srcDir, "db", "migrate")

	return filepath.WalkDir(srcDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if path == migrateDir || entry.Name() == pluginIgnoreDir {
				return filepath.SkipDir
			}
			if rel != "." && ignore.Match(prefix+filepath.ToSlash(rel), true) {
				return filepath.SkipDir
			}
			return nil
		}

		if ignore.Match(prefix+filepath.ToSlash(rel), false) {
			return nil
		}

		rel = strings.TrimSuffix(rel, ".templ")

		dst := filepath.Join(dstRoot, rel)
		if _, err := os.Stat(dst); err == nil {
			fmt.Printf("%s already exists, skipping...\n", rel)
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := ensureDir(filepath.Dir(dst)); err != nil {
			return err
		}

		mode := os.FileMode(0o644)
		if info, err := entry.Info(); err == nil && info.Mode()&0o111 != 0 {
			mode = 0o755
		}

		if err := os.WriteFile(dst, data, mode); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}

		fmt.Printf("Installed %s from plugin %s\n", rel, name)
		return nil
	})
}

// pluginDepsDir is the directory inside a plugin's install/ directory whose
// contents are merged into the host project's deps/ directory by
// `plugin:install` (companion services, deploy configs, ...). It must not be
// named "vendor" — module zips drop vendor/ entirely — and must not contain
// nested go.mod files, which module zips drop as nested modules; ship them as
// go.mod.templ instead, installed as go.mod.
const pluginDepsDir = "deps"

// installPluginDeps merges the plugin's deps/ directory into the host
// project's deps/ directory. The plugin module directory is resolved through
// the host's module graph, so proxy downloads and local replace directives
// behave the same.
func installPluginDeps(name, module string) error {
	dir, err := pluginModuleDir(module)
	if err != nil {
		return fmt.Errorf("locate plugin module %s: %w", module, err)
	}

	return installPluginDepsFrom(name, dir, pluginDepsDir)
}

// pluginModuleDir returns the on-disk directory of a module in the build list
// (module cache for downloads, the checkout itself for replace directives).
func pluginModuleDir(module string) (string, error) {
	list := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", module)
	list.Stderr = os.Stderr

	out, err := list.Output()
	if err != nil {
		return "", err
	}

	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("module %s has no directory", module)
	}

	return dir, nil
}

// installPluginDepsFrom copies every file under the plugin's install/deps/
// directory into dstRoot (the host project's deps/ directory), preserving
// relative paths and stripping a single .templ suffix (so a plugin can ship
// go.mod.templ without tripping Go's nested-module rules, and a future install
// can render such files as templates). A bare file shipped beside its .templ
// variant (go.mod next to go.mod.templ, so a nested module still builds in the
// plugin checkout) is skipped, matching what a module-zip download would
// deliver — the .templ variant is the distribution copy — and a bare go.mod
// that has drifted from its .templ variant fails the install (only
// local-directory installs can hit that: module zips drop the bare file). Any
// ignore/ directory under install/deps/ is skipped wholesale, like the host
// installer, and the plugin's root .gitignore rules exclude matching paths.
// Existing destination files are left untouched, mirroring the
// migration installer's skip behavior. A plugin without an install/deps/
// directory installs nothing.
func installPluginDepsFrom(name, moduleDir, dstRoot string) error {
	srcDir := filepath.Join(moduleDir, pluginInstallDir, pluginDepsDir)
	if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
		return nil
	}

	ignore, err := loadPluginGitIgnore(moduleDir)
	if err != nil {
		return err
	}
	prefix := pluginInstallDir + "/" + pluginDepsDir + "/"

	// Hosts scaffolded by older Airway versions may not have a deps/
	// directory; create it even when the plugin's deps/ holds nothing
	// but .keep, so the host always ends up with one.
	if err := ensureDir(dstRoot); err != nil {
		return err
	}

	// Bare go.mod files verified in sync with their .templ twin: when the
	// .templ variant later lands on an existing destination, the skip is
	// silent — nothing changed since the previous install.
	inSync := map[string]bool{}

	return filepath.WalkDir(srcDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if entry.Name() == pluginIgnoreDir {
				return filepath.SkipDir
			}
			if rel != "." && ignore.Match(prefix+filepath.ToSlash(rel), true) {
				return filepath.SkipDir
			}
			return nil
		}

		if ignore.Match(prefix+filepath.ToSlash(rel), false) {
			return nil
		}

		// The scaffolded deps/.keep only holds the plugin's own empty dir;
		// the host project ships its own deps/.keep.
		if rel == ".keep" {
			return nil
		}
		// A bare file beside its .templ variant is local-development only;
		// installing both would collide on the same destination, and the
		// bare file never survives a module-zip download anyway. A bare
		// go.mod that drifted from its .templ variant means the nested
		// module builds against something else than what ships — fail
		// the install instead of silently preferring the .templ content.
		if _, err := os.Stat(path + ".templ"); err == nil {
			if filepath.Base(path) == "go.mod" {
				bare, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				templ, err := os.ReadFile(path + ".templ")
				if err != nil {
					return err
				}
				if !bytes.Equal(bare, templ) {
					return fmt.Errorf("plugin %s: %s and %s.templ differ; keep them in sync (the .templ variant is what gets installed)", name, rel, rel)
				}
				inSync[rel] = true
			}
			fmt.Printf("%s has a .templ variant, installing that instead\n", rel)
			return nil
		}
		rel = strings.TrimSuffix(rel, ".templ")

		dst := filepath.Join(dstRoot, rel)
		if _, err := os.Stat(dst); err == nil {
			if !inSync[rel] {
				fmt.Printf("%s already exists, skipping...\n", rel)
			}
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := ensureDir(filepath.Dir(dst)); err != nil {
			return err
		}

		mode := os.FileMode(0o644)
		if info, err := entry.Info(); err == nil && info.Mode()&0o111 != 0 {
			mode = 0o755
		}

		if err := os.WriteFile(dst, data, mode); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}

		fmt.Printf("Installed %s from plugin %s\n", rel, name)
		return nil
	})
}
