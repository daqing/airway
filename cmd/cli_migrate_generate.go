package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const migrationUpTemplate = `-- Migration: {{.Slug}} (up)
-- Write the SQL to apply below.
{{.UpExample}}
`

const migrationDownTemplate = `-- Migration: {{.Slug}} (down)
-- Write the SQL to roll back below.
{{.DownExample}}
`

type migrationTemplateData struct {
	Version     string
	Name        string
	Slug        string
	UpExample   string
	DownExample string
}

func generateMigrationFiles(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIGenerateMigrationUsage(os.Stdout)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: airway generate migration [name]")
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("migration name must not be empty")
	}

	if err := ensureDir(migrationDir); err != nil {
		return err
	}

	version := timeNow().Format("20060102150405")
	data := migrationTemplateData{
		Version:     version,
		Name:        strings.TrimPrefix(name, "create_"),
		Slug:        name,
		UpExample:   "-- e.g. CREATE TABLE ...;",
		DownExample: "-- e.g. DROP TABLE ...;",
	}

	if table, ok := strings.CutPrefix(name, "create_"); ok && table != "" {
		data.UpExample = fmt.Sprintf(`--
-- CREATE TABLE %s (
--   id BIGINT PRIMARY KEY,
--   created_at TIMESTAMP NOT NULL,
--   updated_at TIMESTAMP NOT NULL
-- );`, table)
		data.DownExample = fmt.Sprintf("--\n-- DROP TABLE %s;", table)
	}

	upPath := filepath.Join(migrationDir, fmt.Sprintf("%s_%s.up.sql", version, name))
	if err := writeTemplateFile(migrationUpTemplate, upPath, data); err != nil {
		return err
	}

	downPath := filepath.Join(migrationDir, fmt.Sprintf("%s_%s.down.sql", version, name))
	if err := writeTemplateFile(migrationDownTemplate, downPath, data); err != nil {
		return err
	}

	fmt.Printf("Created migration %s\n", upPath)
	fmt.Printf("Created migration %s\n", downPath)
	return nil
}
