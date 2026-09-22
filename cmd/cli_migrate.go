package cmd

import (
	"os"
	"strings"

	"github.com/daqing/airway/lib/migrate"
)

const migrationDir = "./db/migrate"
const schemaSnapshotPath = "./db/schema.json"

func runCLIMigrate(args []string) error {
	version := ""
	if len(args) > 0 {
		version = strings.TrimSpace(args[0])
	}

	opts, err := cliMigrationOptions()
	if err != nil {
		return err
	}

	return migrate.RunTo(opts, version)
}

func runCLIRollback(args []string) error {
	step := 1
	if len(args) > 0 {
		var err error
		step, err = parsePositiveInt(args[0])
		if err != nil {
			return err
		}
	}

	opts, err := cliMigrationOptions()
	if err != nil {
		return err
	}

	return migrate.Rollback(opts, step)
}

func runCLIStatus(_ []string) error {
	opts, err := cliMigrationOptions()
	if err != nil {
		return err
	}

	return migrate.Status(opts)
}

// cliMigrationOptions points the shared migrate engine at the project's
// on-disk migration directory and schema snapshot.
func cliMigrationOptions() (migrate.Options, error) {
	dsn, err := cliDSN()
	if err != nil {
		return migrate.Options{}, err
	}

	return migrate.Options{
		DSN:          dsn,
		Migrations:   os.DirFS(migrationDir),
		SnapshotPath: schemaSnapshotPath,
	}, nil
}
