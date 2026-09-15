package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	_, _ = fmt.Fprintln(w, "  airway plugin:install <module>")
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

// runCLIPluginInstall copies a plugin's embedded SQL migrations into the
// host's db/migrate directory with fresh timestamps, so they run through the
// regular db:migrate / db:rollback / db:status machinery. The argument is the
// plugin's module path (e.g. github.com/daqing/airway-im-plugin); the plugin
// name is derived from its last segment, same as `plugin:new`.
func runCLIPluginInstall(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: airway plugin:install <module>")
	}

	module := strings.TrimSpace(args[0])
	name := pluginNameFromModule(module)
	if !pluginNamePattern.MatchString(name) {
		return fmt.Errorf("cannot derive a plugin name from %q; use the plugin's module path (e.g. github.com/daqing/airway-im-plugin)", module)
	}

	p := plugin.Find(name)
	if p == nil {
		return fmt.Errorf("plugin %q (%s) is not registered; run `go get %s`, add its blank import to plugins.go, and retry with the project binary", name, module, module)
	}

	provider, ok := p.(plugin.MigrationProvider)
	if !ok {
		fmt.Printf("Plugin %s has no SQL migrations to install\n", name)
		return nil
	}

	return installPluginMigrations(name, provider.MigrationFS(), migrationDir, timeNow())
}

func installPluginMigrations(pluginName string, migrations fs.FS, dstDir string, now time.Time) error {
	pairs, err := collectPluginMigrationPairs(migrations)
	if err != nil {
		return fmt.Errorf("read plugin %s migrations: %w", pluginName, err)
	}

	if len(pairs) == 0 {
		fmt.Printf("Plugin %s has no SQL migrations to install\n", pluginName)
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

func collectPluginMigrationPairs(migrations fs.FS) ([]pluginMigrationPair, error) {
	ups := map[string][]byte{}
	downs := map[string][]byte{}

	err := fs.WalkDir(migrations, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
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
