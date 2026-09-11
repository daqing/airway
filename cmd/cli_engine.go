package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/daqing/airway/lib/engine"
)

func runCLIEngine(args []string) error {
	if len(args) == 0 {
		printCLIEngineUsage(os.Stdout)
		return nil
	}

	subcommand := strings.ToLower(strings.TrimSpace(args[0]))

	switch subcommand {
	case "list":
		return runCLIEngineList()
	case "install":
		return runCLIEngineInstall(args[1:])
	case "help", "-h", "--help":
		printCLIEngineUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown engine command: %s", subcommand)
	}
}

func printCLIEngineUsage(w *os.File) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway engine list")
	_, _ = fmt.Fprintln(w, "  airway engine install [name]")
}

func runCLIEngineList() error {
	engines := engine.Engines()
	if len(engines) == 0 {
		fmt.Println("No engines registered (add blank imports to engines.go)")
		return nil
	}

	for _, e := range engines {
		fmt.Printf("%s\t%s\n", e.Name(), e.MountPath())
	}

	return nil
}

// runCLIEngineInstall copies an engine's embedded SQL migrations into the
// host's db/migrate directory with fresh timestamps, so they run through the
// regular db:migrate / db:rollback / db:status machinery.
func runCLIEngineInstall(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: airway engine install [name]")
	}

	name := strings.TrimSpace(args[0])

	e := engine.Find(name)
	if e == nil {
		return fmt.Errorf("engine %q is not registered; add its blank import to engines.go", name)
	}

	provider, ok := e.(engine.MigrationProvider)
	if !ok {
		fmt.Printf("Engine %s has no SQL migrations to install\n", name)
		return nil
	}

	return installEngineMigrations(name, provider.MigrationFS(), migrationDir, timeNow())
}

func installEngineMigrations(engineName string, migrations fs.FS, dstDir string, now time.Time) error {
	pairs, err := collectEngineMigrationPairs(migrations)
	if err != nil {
		return fmt.Errorf("read engine %s migrations: %w", engineName, err)
	}

	if len(pairs) == 0 {
		fmt.Printf("Engine %s has no SQL migrations to install\n", engineName)
		return nil
	}

	if err := ensureDir(dstDir); err != nil {
		return err
	}

	for i, pair := range pairs {
		version := now.Add(time.Duration(i) * time.Second).Format("20060102150405")

		if engineMigrationInstalled(dstDir, pair.name) {
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

		fmt.Printf("Installed migration %s_%s from engine %s\n", version, pair.name, engineName)
	}

	return nil
}

type engineMigrationPair struct {
	name string
	up   []byte
	down []byte
}

func collectEngineMigrationPairs(migrations fs.FS) ([]engineMigrationPair, error) {
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
		// Strip the engine's own version prefix; the install assigns a fresh one.
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

	pairs := make([]engineMigrationPair, 0, len(names))
	for _, name := range names {
		pairs = append(pairs, engineMigrationPair{name: name, up: ups[name], down: downs[name]})
	}

	return pairs, nil
}

// engineMigrationInstalled reports whether db/migrate already contains a
// migration whose name part matches (regardless of its timestamp prefix).
func engineMigrationInstalled(dstDir string, name string) bool {
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
