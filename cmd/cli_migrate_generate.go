package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type migrationTemplateData struct {
	Version string
	Name    string
	Slug    string
}

const migrationTemplate = `package migrate

import "github.com/daqing/airway/lib/migrate/schema"

func init() {
	schema.RegisterChange("{{.Version}}", "{{.Slug}}", func(m *schema.Migrator) {
		m.CreateTable("{{.Name}}", func(t *schema.Table) {
			t.ID()
			t.Timestamps()
		})
	})
}
`

func generateMigrationFiles(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIGenerateMigrationUsage(os.Stdout)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: airway cli generate migration [name]")
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("migration name must not be empty")
	}

	if err := ensureDir(migrationDir); err != nil {
		return err
	}

	version := timeNow().Format("20060102150405")
	targetPath := filepath.Join(migrationDir, fmt.Sprintf("%s_%s.go", version, name))

	return writeTemplateFile(migrationTemplate, targetPath, migrationTemplateData{
		Version: version,
		Name:    strings.TrimPrefix(strings.TrimSpace(name), "create_"),
		Slug:    name,
	})
}
