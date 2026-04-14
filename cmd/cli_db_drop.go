package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/daqing/airway/lib/repo"
)

func runCLIDBDrop(_ []string) error {
	dsn, err := cliDSN()
	if err != nil {
		return err
	}

	driver, dbName, adminDSN, err := parseDBConfig(dsn)
	if err != nil {
		return err
	}

	if driver == repo.DriverSQLite {
		if err := os.Remove(dbName); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Database file %s does not exist, skipping...\n", dbName)
				return nil
			}
			return fmt.Errorf("remove database file: %w", err)
		}
		fmt.Printf("Dropped database: %s\n", dbName)
		return nil
	}

	db, err := repo.NewDBWithDriver(string(driver), adminDSN)
	if err != nil {
		return fmt.Errorf("connect to admin database: %w", err)
	}
	defer db.Close()

	_, err = db.Conn().ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		return fmt.Errorf("drop database: %w", err)
	}

	fmt.Printf("Dropped database: %s\n", dbName)
	return nil
}

func parseDBConfig(dsn string) (repo.Driver, string, string, error) {
	driver := detectDriverForDrop(dsn)
	switch driver {
	case repo.DriverPostgres:
		return parsePostgresDBDrop(dsn)
	case repo.DriverMySQL:
		return parseMySQLDBDrop(dsn)
	case repo.DriverSQLite:
		return driver, normalizeSQLitePath(dsn), "", nil
	default:
		return "", "", "", fmt.Errorf("unsupported database driver for db:drop")
	}
}

func detectDriverForDrop(dsn string) repo.Driver {
	lowerDSN := strings.ToLower(strings.TrimSpace(dsn))
	switch {
	case strings.HasPrefix(lowerDSN, "postgres://"), strings.HasPrefix(lowerDSN, "postgresql://"):
		return repo.DriverPostgres
	case strings.HasPrefix(lowerDSN, "mysql://"):
		return repo.DriverMySQL
	case strings.HasPrefix(lowerDSN, "sqlite://"), strings.HasPrefix(lowerDSN, "sqlite3://"):
		return repo.DriverSQLite
	case strings.HasPrefix(lowerDSN, "file:"), lowerDSN == ":memory:":
		return repo.DriverSQLite
	case strings.Contains(lowerDSN, "@tcp("), strings.Contains(lowerDSN, "@unix("), strings.Contains(lowerDSN, ")/"):
		return repo.DriverMySQL
	case strings.Contains(lowerDSN, "host="), strings.Contains(lowerDSN, "user="), strings.Contains(lowerDSN, "dbname="):
		return repo.DriverPostgres
	case strings.HasSuffix(lowerDSN, ".db"), strings.HasSuffix(lowerDSN, ".sqlite"), strings.HasSuffix(lowerDSN, ".sqlite3"):
		return repo.DriverSQLite
	default:
		return ""
	}
}

func normalizeSQLitePath(dsn string) string {
	lowerDSN := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lowerDSN, "sqlite://"):
		return dsn[len("sqlite://"):]
	case strings.HasPrefix(lowerDSN, "sqlite3://"):
		return dsn[len("sqlite3://"):]
	default:
		return dsn
	}
}

func parsePostgresDBDrop(dsn string) (repo.Driver, string, string, error) {
	lowerDSN := strings.ToLower(strings.TrimSpace(dsn))
	if strings.HasPrefix(lowerDSN, "postgres://") || strings.HasPrefix(lowerDSN, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", "", "", fmt.Errorf("parse postgres dsn: %w", err)
		}
		dbName := strings.TrimPrefix(u.Path, "/")
		if dbName == "" {
			return "", "", "", fmt.Errorf("database name not found in postgres dsn")
		}
		adminURL := *u
		adminURL.Path = "/postgres"
		return repo.DriverPostgres, dbName, adminURL.String(), nil
	}

	// key-value format: host=... dbname=... user=...
	fields := strings.Fields(dsn)
	dbName := ""
	newFields := make([]string, 0, len(fields))
	for _, f := range fields {
		if strings.HasPrefix(f, "dbname=") {
			dbName = strings.TrimPrefix(f, "dbname=")
			continue
		}
		newFields = append(newFields, f)
	}
	if dbName == "" {
		return "", "", "", fmt.Errorf("database name not found in postgres dsn")
	}
	newFields = append(newFields, "dbname=postgres")
	return repo.DriverPostgres, dbName, strings.Join(newFields, " "), nil
}

func parseMySQLDBDrop(dsn string) (repo.Driver, string, string, error) {
	lowerDSN := strings.ToLower(strings.TrimSpace(dsn))
	if strings.HasPrefix(lowerDSN, "mysql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", "", "", fmt.Errorf("parse mysql dsn: %w", err)
		}
		dbName := strings.TrimPrefix(u.Path, "/")
		if dbName == "" {
			return "", "", "", fmt.Errorf("database name not found in mysql dsn")
		}
		adminURL := *u
		adminURL.Path = "/"
		return repo.DriverMySQL, dbName, adminURL.String(), nil
	}

	// standard format: user:pass@tcp(host:port)/dbname?params
	qIdx := strings.Index(dsn, "?")
	before := dsn
	after := ""
	if qIdx >= 0 {
		before = dsn[:qIdx]
		after = dsn[qIdx:]
	}
	slashIdx := strings.LastIndex(before, "/")
	if slashIdx < 0 {
		return "", "", "", fmt.Errorf("database name not found in mysql dsn")
	}
	dbName := before[slashIdx+1:]
	adminDSN := before[:slashIdx+1] + after
	return repo.DriverMySQL, dbName, adminDSN, nil
}
