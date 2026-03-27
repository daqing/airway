package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/daqing/airway/lib/repo"
	"github.com/jmoiron/sqlx"
)

const migrationDir = "./db/migrate"
const schemaMigrationsTable = "schema_migrations"

type migrationFile struct {
	Version string
	Name    string
	Path    string
}

func runCLIMigrate(args []string) error {
	version := ""
	if len(args) > 0 {
		version = strings.TrimSpace(args[0])
	}

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		return err
	}
	defer manager.Close()

	return manager.Migrate(version)
}

func runCLIRollback(args []string) error {
	step := 1
	if len(args) > 0 {
		var err error
		step, err = parsePositiveInt(args[0])
		if err != nil {
			return fmt.Errorf("invalid rollback step %q: %w", args[0], err)
		}
	}

	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		return err
	}
	defer manager.Close()

	return manager.Rollback(step)
}

func runCLIStatus(_ []string) error {
	manager, err := newMigrationManagerFromEnv()
	if err != nil {
		return err
	}
	defer manager.Close()

	return manager.Status()
}

type migrationManager struct {
	db  *repo.DB
	now func() time.Time
}

func newMigrationManagerFromEnv() (*migrationManager, error) {
	dsn, err := cliDSN()
	if err != nil {
		return nil, err
	}

	db, err := repo.NewDB(dsn)
	if err != nil {
		return nil, err
	}

	return &migrationManager{db: db, now: time.Now}, nil
}

func (m *migrationManager) Close() error {
	if m == nil || m.db == nil {
		return nil
	}

	return m.db.Close()
}

func (m *migrationManager) Migrate(targetVersion string) error {
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	files, err := readMigrationFiles(migrationDir, ".up.sql")
	if err != nil {
		return err
	}

	applied, err := m.appliedVersions()
	if err != nil {
		return err
	}

	pending := make([]migrationFile, 0, len(files))
	for _, file := range files {
		if applied[file.Version] {
			fmt.Printf("Migration %s already applied, skipping...\n", filepath.Base(file.Path))
			continue
		}

		if targetVersion != "" && file.Version > targetVersion {
			continue
		}

		pending = append(pending, file)
	}

	if len(pending) == 0 {
		fmt.Printf("Already at the latest migration\n")
		return nil
	}

	for _, file := range pending {
		if err := m.applyMigration(file); err != nil {
			return err
		}
	}

	fmt.Printf("Migration files inside %s executed successfully\n", migrationDir)
	return nil
}

func (m *migrationManager) Rollback(step int) error {
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	if step <= 0 {
		step = 1
	}

	applied, err := m.appliedVersionList()
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		fmt.Printf("Already at the latest migration\n")
		return nil
	}

	if step > len(applied) {
		step = len(applied)
	}

	downFiles, err := readMigrationFiles(migrationDir, ".down.sql")
	if err != nil {
		return err
	}

	downByVersion := make(map[string]migrationFile, len(downFiles))
	for _, file := range downFiles {
		downByVersion[file.Version] = file
	}

	for i := 0; i < step; i++ {
		version := applied[len(applied)-1-i]
		file, ok := downByVersion[version]
		if !ok {
			return fmt.Errorf("rollback file for version %s not found", version)
		}

		if err := m.rollbackMigration(file); err != nil {
			return err
		}
	}

	return nil
}

func (m *migrationManager) Status() error {
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	upFiles, err := readMigrationFiles(migrationDir, ".up.sql")
	if err != nil {
		return err
	}

	applied, err := m.appliedVersionSet()
	if err != nil {
		return err
	}

	if len(upFiles) == 0 {
		fmt.Println("No migration files found")
		return nil
	}

	for _, file := range upFiles {
		state := "pending"
		if applied[file.Version] {
			state = "applied"
		}

		fmt.Printf("%s\t%s\n", state, filepath.Base(file.Path))
	}

	return nil
}

func (m *migrationManager) ensureMigrationTable() error {
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	version VARCHAR(255) PRIMARY KEY,
	applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`, schemaMigrationsTable)

	_, err := m.db.Conn().ExecContext(context.Background(), query)
	return err
}

func (m *migrationManager) appliedVersions() (map[string]bool, error) {
	list, err := m.appliedVersionList()
	if err != nil {
		return nil, err
	}

	set := make(map[string]bool, len(list))
	for _, version := range list {
		set[version] = true
	}

	return set, nil
}

func (m *migrationManager) appliedVersionSet() (map[string]bool, error) {
	return m.appliedVersions()
}

func (m *migrationManager) appliedVersionList() ([]string, error) {
	rows, err := m.db.Conn().QueryxContext(
		context.Background(),
		fmt.Sprintf("SELECT version FROM %s ORDER BY version ASC", schemaMigrationsTable),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := []string{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}

		versions = append(versions, version)
	}

	return versions, rows.Err()
}

func (m *migrationManager) applyMigration(file migrationFile) error {
	sqlText, err := os.ReadFile(file.Path)
	if err != nil {
		return err
	}

	fmt.Printf("Running migration file %s...\n", filepath.Base(file.Path))

	tx, err := m.db.Conn().BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := execSQLScript(context.Background(), tx, string(sqlText)); err != nil {
		return fmt.Errorf("execute %s: %w", filepath.Base(file.Path), err)
	}

	if _, err := tx.ExecContext(
		context.Background(),
		fmt.Sprintf("INSERT INTO %s (version) VALUES (%s)", schemaMigrationsTable, m.bindVar(1)),
		file.Version,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (m *migrationManager) rollbackMigration(file migrationFile) error {
	sqlText, err := os.ReadFile(file.Path)
	if err != nil {
		return err
	}

	tx, err := m.db.Conn().BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := execSQLScript(context.Background(), tx, string(sqlText)); err != nil {
		return fmt.Errorf("execute %s: %w", filepath.Base(file.Path), err)
	}

	if _, err := tx.ExecContext(
		context.Background(),
		fmt.Sprintf("DELETE FROM %s WHERE version = %s", schemaMigrationsTable, m.bindVar(1)),
		file.Version,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("Migration %s rolled back successfully\n", file.Version)
	return nil
}

func readMigrationFiles(dir string, suffix string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}

		version, err := extractMigrationVersion(entry.Name(), suffix)
		if err != nil {
			return nil, err
		}

		files = append(files, migrationFile{
			Version: version,
			Name:    entry.Name(),
			Path:    filepath.Join(dir, entry.Name()),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Version < files[j].Version
	})

	return files, nil
}

func extractMigrationVersion(fileName string, suffix string) (string, error) {
	if !strings.HasSuffix(fileName, suffix) {
		return "", fmt.Errorf("invalid migration file %s", fileName)
	}

	base := strings.TrimSuffix(fileName, suffix)
	version, _, found := strings.Cut(base, "_")
	if !found || strings.TrimSpace(version) == "" {
		return "", fmt.Errorf("migration file %s must start with timestamp prefix", fileName)
	}

	return version, nil
}

func execSQLScript(ctx context.Context, tx *sqlx.Tx, script string) error {
	statements := splitSQLStatements(script)
	for _, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}

		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

func splitSQLStatements(script string) []string {
	var statements []string
	var current strings.Builder

	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(script); i++ {
		ch := script[i]
		next := byte(0)
		if i+1 < len(script) {
			next = script[i+1]
		}

		if inLineComment {
			current.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}

		if inBlockComment {
			current.WriteByte(ch)
			if ch == '*' && next == '/' {
				current.WriteByte(next)
				i++
				inBlockComment = false
			}
			continue
		}

		if !inSingleQuote && !inDoubleQuote && !inBacktick {
			if ch == '-' && next == '-' {
				current.WriteByte(ch)
				current.WriteByte(next)
				i++
				inLineComment = true
				continue
			}

			if ch == '/' && next == '*' {
				current.WriteByte(ch)
				current.WriteByte(next)
				i++
				inBlockComment = true
				continue
			}
		}

		switch ch {
		case '\'':
			if !inDoubleQuote && !inBacktick && !isEscaped(script, i) {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote && !inBacktick && !isEscaped(script, i) {
				inDoubleQuote = !inDoubleQuote
			}
		case '`':
			if !inSingleQuote && !inDoubleQuote {
				inBacktick = !inBacktick
			}
		case ';':
			if !inSingleQuote && !inDoubleQuote && !inBacktick {
				trimmed := strings.TrimSpace(current.String())
				if trimmed != "" {
					statements = append(statements, trimmed)
				}
				current.Reset()
				continue
			}
		}

		current.WriteByte(ch)
	}

	if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
		statements = append(statements, trimmed)
	}

	return statements
}

func isEscaped(value string, index int) bool {
	backslashes := 0
	for i := index - 1; i >= 0 && value[i] == '\\'; i-- {
		backslashes++
	}

	return backslashes%2 == 1
}

func (m *migrationManager) bindVar(position int) string {
	if m.db.Driver() == repo.DriverPostgres {
		return fmt.Sprintf("$%d", position)
	}

	return "?"
}
