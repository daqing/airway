package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func generateMigrationFiles(args []string) error {
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

	timestamp := timeNow().Format("20060102150405")
	upPath := filepath.Join(migrationDir, fmt.Sprintf("%s_%s.up.sql", timestamp, name))
	downPath := filepath.Join(migrationDir, fmt.Sprintf("%s_%s.down.sql", timestamp, name))

	if err := writeNewFile(upPath, "-- Write your up migration here.\n"); err != nil {
		return err
	}

	if err := writeNewFile(downPath, "-- Write your down migration here.\n"); err != nil {
		return err
	}

	fmt.Printf("Created %s\n", upPath)
	fmt.Printf("Created %s\n", downPath)
	return nil
}

func writeNewFile(path string, content string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}
