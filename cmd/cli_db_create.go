package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/daqing/airway/lib/repo"
)

func runCLIDBCreate(_ []string) error {
	dsn, err := cliDSN()
	if err != nil {
		return err
	}

	driver, dbName, adminDSN, err := parseDBConfig(dsn)
	if err != nil {
		return err
	}

	if driver == repo.DriverSQLite {
		_, err := os.Stat(dbName)
		if err == nil {
			fmt.Printf("Database file %s already exists, skipping...\n", dbName)
			return nil
		}
		f, err := os.Create(dbName)
		if err != nil {
			return fmt.Errorf("create database file: %w", err)
		}
		_ = f.Close()
		fmt.Printf("Created database: %s\n", dbName)
		return nil
	}

	db, err := repo.NewDBWithDriver(string(driver), adminDSN)
	if err != nil {
		return fmt.Errorf("connect to admin database: %w", err)
	}
	defer db.Close()

	_, err = db.Conn().ExecContext(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	fmt.Printf("Created database: %s\n", dbName)
	return nil
}
