package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/daqing/airway/db/migrate"
	"github.com/daqing/airway/lib/migrate/dialect"
	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
	"github.com/jmoiron/sqlx"
)

const migrationDir = "./db/migrate"
const schemaMigrationsTable = "schema_migrations"
const schemaSnapshotPath = "./db/schema.json"

type migrationFile struct {
	Version string
	Name    string
	Path    string
}

type migrationUnit struct {
	Version string
	Name    string
	Kind    string
	UpSQL   string
	DownSQL string
	UpOps   []schema.Operation
	DownOps []schema.Operation
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
	db       *repo.DB
	now      func() time.Time
	compiler *dialect.Compiler
	state    *schema.State
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

	return &migrationManager{db: db, now: time.Now, compiler: dialect.NewCompiler(db.Driver())}, nil
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

	units, err := m.loadMigrationUnits()
	if err != nil {
		return err
	}

	applied, err := m.appliedVersions()
	if err != nil {
		return err
	}

	m.state, _ = deriveSchemaState(units, applied)
	m.restoreSchemaState()

	pending := make([]migrationUnit, 0, len(units))
	for _, unit := range units {
		if applied[unit.Version] {
			fmt.Printf("Migration %s already applied, skipping...\n", unit.displayName())
			continue
		}

		if targetVersion != "" && unit.Version > targetVersion {
			continue
		}

		pending = append(pending, unit)
	}

	if len(pending) == 0 {
		fmt.Printf("Already at the latest migration\n")
	} else {
		for _, unit := range pending {
			if err := m.applyMigration(unit); err != nil {
				return err
			}
			if unit.Kind == "sql" {
				m.state = nil
			}
		}
		fmt.Printf("Migration files inside %s executed successfully\n", migrationDir)
	}

	if err := m.dumpSchemaState(); err != nil {
		return err
	}

	return nil
}

func (m *migrationManager) dumpSchemaState() error {
	state, err := repo.InspectSchema(m.db)
	if err != nil {
		return fmt.Errorf("inspect schema: %w", err)
	}

	if err := schema.SaveSnapshot(schemaSnapshotPath, state, true); err != nil {
		return fmt.Errorf("save schema snapshot: %w", err)
	}

	fmt.Printf("Schema snapshot written to %s\n", schemaSnapshotPath)
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

	units, err := m.loadMigrationUnits()
	if err != nil {
		return err
	}

	appliedSet := make(map[string]bool, len(applied))
	for _, version := range applied {
		appliedSet[version] = true
	}
	m.state, _ = deriveSchemaState(units, appliedSet)
	m.restoreSchemaState()

	downByVersion := make(map[string]migrationUnit, len(units))
	for _, unit := range units {
		downByVersion[unit.Version] = unit
	}

	for i := 0; i < step; i++ {
		version := applied[len(applied)-1-i]
		unit, ok := downByVersion[version]
		if !ok {
			return fmt.Errorf("rollback file for version %s not found", version)
		}

		if err := m.rollbackMigration(unit); err != nil {
			return err
		}
		if unit.Kind == "sql" {
			m.state = nil
		}
	}

	if err := m.dumpSchemaState(); err != nil {
		return err
	}

	return nil
}

func (m *migrationManager) Status() error {
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	units, err := m.loadMigrationUnits()
	if err != nil {
		return err
	}

	applied, err := m.appliedVersionSet()
	if err != nil {
		return err
	}

	if len(units) == 0 {
		fmt.Println("No migration files found")
		return nil
	}

	for _, unit := range units {
		state := "pending"
		if applied[unit.Version] {
			state = "applied"
		}

		fmt.Printf("%s\t%s\n", state, unit.displayName())
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

func (m *migrationManager) applyMigration(unit migrationUnit) error {
	fmt.Printf("Running migration file %s...\n", unit.displayName())

	tx, err := m.db.Conn().BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := m.executeMigrationUnit(context.Background(), tx, unit, true); err != nil {
		return fmt.Errorf("execute %s: %w", unit.displayName(), err)
	}

	if _, err := tx.ExecContext(
		context.Background(),
		fmt.Sprintf("INSERT INTO %s (version) VALUES (%s)", schemaMigrationsTable, m.bindVar(1)),
		unit.Version,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (m *migrationManager) rollbackMigration(unit migrationUnit) error {
	tx, err := m.db.Conn().BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := m.executeMigrationUnit(context.Background(), tx, unit, false); err != nil {
		return fmt.Errorf("execute %s: %w", unit.displayName(), err)
	}

	if _, err := tx.ExecContext(
		context.Background(),
		fmt.Sprintf("DELETE FROM %s WHERE version = %s", schemaMigrationsTable, m.bindVar(1)),
		unit.Version,
	); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("Migration %s rolled back successfully\n", unit.Version)
	return nil
}

func (m *migrationManager) executeMigrationUnit(ctx context.Context, tx *sqlx.Tx, unit migrationUnit, up bool) error {
	switch unit.Kind {
	case "sql":
		sqlText := unit.UpSQL
		if !up {
			sqlText = unit.DownSQL
		}
		return execSQLScript(ctx, tx, sqlText)
	case "dsl":
		ops := unit.UpOps
		if !up {
			ops = unit.DownOps
		}
		savedState := m.state
		workingState := savedState
		if savedState != nil {
			workingState = savedState.Clone()
		}
		m.state = workingState
		err := m.executeOperations(ctx, tx, ops)
		if err != nil {
			m.state = savedState
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported migration kind: %s", unit.Kind)
	}
}

func (m *migrationManager) executeOperations(ctx context.Context, tx *sqlx.Tx, ops []schema.Operation) error {
	for _, op := range ops {
		resolvedOp, err := m.resolveOperation(op)
		if err != nil {
			return err
		}

		if m.db.Driver() == repo.DriverSQLite {
			if handled, err := m.executeSQLiteSchemaOperation(ctx, tx, resolvedOp); handled || err != nil {
				if err != nil {
					return err
				}
				if m.state != nil {
					if err := m.state.Apply(resolvedOp); err != nil {
						return err
					}
				}
				continue
			}
		}

		statements, err := m.compiler.Compile(resolvedOp)
		if err != nil {
			return err
		}

		for _, stmt := range statements {
			if strings.TrimSpace(stmt) == "" {
				continue
			}

			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("%s: %w", stmt, err)
			}
		}

		if m.state != nil {
			if err := m.state.Apply(resolvedOp); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *migrationManager) resolveOperation(op schema.Operation) (schema.Operation, error) {
	if m.state == nil {
		return op, nil
	}

	switch actual := op.(type) {
	case schema.SetNullOp:
		column, ok := m.state.Column(actual.Table, actual.ColumnName)
		if !ok {
			return nil, fmt.Errorf("column %s does not exist on table %s", actual.ColumnName, actual.Table)
		}
		column.Null = schema.Bool(actual.Nullable)
		return schema.ChangeColumnOp{Table: actual.Table, Column: column}, nil
	case schema.SetDefaultOp:
		column, ok := m.state.Column(actual.Table, actual.ColumnName)
		if !ok {
			return nil, fmt.Errorf("column %s does not exist on table %s", actual.ColumnName, actual.Table)
		}
		if actual.Remove {
			column.Default = nil
		} else {
			column.Default = actual.Default
		}
		return schema.ChangeColumnOp{Table: actual.Table, Column: column}, nil
	default:
		return op, nil
	}
}

func (m *migrationManager) executeSQLiteSchemaOperation(ctx context.Context, tx *sqlx.Tx, op schema.Operation) (bool, error) {
	if m.state == nil {
		return false, nil
	}

	switch actual := op.(type) {
	case schema.RemoveColumnOp:
		return true, m.executeSQLiteTableRebuild(ctx, tx, actual.Table, actual)
	case schema.ChangeColumnOp:
		return true, m.executeSQLiteTableRebuild(ctx, tx, actual.Table, actual)
	case schema.AddForeignKeyOp:
		return true, m.executeSQLiteTableRebuild(ctx, tx, actual.Table, actual)
	case schema.RemoveForeignKeyOp:
		return true, m.executeSQLiteTableRebuild(ctx, tx, actual.Table, actual)
	default:
		return false, nil
	}
}

func (m *migrationManager) executeSQLiteTableRebuild(ctx context.Context, tx *sqlx.Tx, tableName string, op schema.Operation) error {
	currentTable, ok := m.state.Table(tableName)
	if !ok {
		return fmt.Errorf("table %s does not exist in schema state", tableName)
	}

	nextState := m.state.Clone()
	if err := nextState.Apply(op); err != nil {
		return err
	}

	nextTable, ok := nextState.Table(tableName)
	if !ok {
		return fmt.Errorf("table %s does not exist after applying operation", tableName)
	}

	tempTableName := fmt.Sprintf("__airway_tmp_%s_%s", tableName, m.now().Format("150405"))
	tempTable := nextTable.Clone()
	tempTable.Name = tempTableName

	createSQL, err := m.compiler.Compile(schema.CreateTableOp{
		Table:       tempTable.Name,
		Columns:     tempTable.Columns,
		Indexes:     tempTable.Indexes,
		ForeignKeys: tempTable.ForeignKeys,
	})
	if err != nil {
		return err
	}

	for _, stmt := range createSQL[:1] {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	columns := make([]string, 0, len(nextTable.Columns))
	for _, column := range sharedColumns(currentTable.Columns, nextTable.Columns) {
		columns = append(columns, quoteIdent(column.Name, repo.DriverSQLite))
	}
	columnList := strings.Join(columns, ", ")
	if columnList != "" {
		copySQL := fmt.Sprintf(
			"INSERT INTO %s (%s) SELECT %s FROM %s",
			quoteIdent(tempTableName, repo.DriverSQLite),
			columnList,
			columnList,
			quoteIdent(tableName, repo.DriverSQLite),
		)
		if _, err := tx.ExecContext(ctx, copySQL); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", quoteIdent(tableName, repo.DriverSQLite))); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteIdent(tempTableName, repo.DriverSQLite), quoteIdent(tableName, repo.DriverSQLite))); err != nil {
		return err
	}

	for _, stmt := range createSQL[1:] {
		rewritten := strings.ReplaceAll(stmt, quoteIdent(tempTableName, repo.DriverSQLite), quoteIdent(tableName, repo.DriverSQLite))
		if _, err := tx.ExecContext(ctx, rewritten); err != nil {
			return err
		}
	}

	return nil
}

func (m *migrationManager) loadMigrationUnits() ([]migrationUnit, error) {
	sqlUpFiles, err := readMigrationFiles(migrationDir, ".up.sql")
	if err != nil {
		return nil, err
	}

	sqlDownFiles, err := readMigrationFiles(migrationDir, ".down.sql")
	if err != nil {
		return nil, err
	}

	downByVersion := map[string]migrationFile{}
	for _, file := range sqlDownFiles {
		downByVersion[file.Version] = file
	}

	units := make([]migrationUnit, 0, len(sqlUpFiles)+len(schema.Definitions()))
	versions := map[string]bool{}

	for _, upFile := range sqlUpFiles {
		downFile, ok := downByVersion[upFile.Version]
		if !ok {
			return nil, fmt.Errorf("missing down migration for version %s", upFile.Version)
		}

		upSQL, err := os.ReadFile(upFile.Path)
		if err != nil {
			return nil, err
		}

		downSQL, err := os.ReadFile(downFile.Path)
		if err != nil {
			return nil, err
		}

		if versions[upFile.Version] {
			return nil, fmt.Errorf("duplicate migration version %s", upFile.Version)
		}
		versions[upFile.Version] = true

		units = append(units, migrationUnit{
			Version: upFile.Version,
			Name:    upFile.Name,
			Kind:    "sql",
			UpSQL:   string(upSQL),
			DownSQL: string(downSQL),
		})
	}

	for _, def := range schema.Definitions() {
		if versions[def.Version] {
			return nil, fmt.Errorf("duplicate migration version %s", def.Version)
		}
		versions[def.Version] = true

		units = append(units, migrationUnit{
			Version: def.Version,
			Name:    def.Name + ".go",
			Kind:    "dsl",
			UpOps:   def.UpOps,
			DownOps: def.DownOps,
		})
	}

	sort.Slice(units, func(i, j int) bool {
		return units[i].Version < units[j].Version
	})

	return units, nil
}

func deriveSchemaState(units []migrationUnit, applied map[string]bool) (*schema.State, bool) {
	state := schema.NewState()
	for _, unit := range units {
		if !applied[unit.Version] {
			continue
		}
		if unit.Kind != "dsl" {
			return nil, false
		}
		if err := state.ApplyAll(unit.UpOps); err != nil {
			return nil, false
		}
	}
	return state, true
}

func (m *migrationManager) restoreSchemaState() {
	snapshot, err := schema.LoadSnapshot(schemaSnapshotPath)
	if err != nil {
		return
	}

	if snapshot.Known && snapshot.State != nil {
		m.state = snapshot.State.Clone()
	}
}

func sharedColumns(current []schema.Column, next []schema.Column) []schema.Column {
	currentByName := make(map[string]schema.Column, len(current))
	for _, column := range current {
		currentByName[column.Name] = column
	}

	shared := make([]schema.Column, 0, len(next))
	for _, column := range next {
		if _, ok := currentByName[column.Name]; ok {
			shared = append(shared, column)
		}
	}

	return shared
}

func quoteIdent(name string, driver repo.Driver) string {
	switch driver {
	case repo.DriverMySQL:
		return "`" + strings.TrimSpace(name) + "`"
	default:
		return `"` + strings.TrimSpace(name) + `"`
	}
}

func (u migrationUnit) displayName() string {
	if strings.TrimSpace(u.Name) == "" {
		return u.Version
	}

	return u.Name
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
	statements, err := dialect.NewCompiler(repo.DriverSQLite).Compile(schema.RawSQLOp{UpSQL: script})
	if err != nil {
		return err
	}

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

func (m *migrationManager) bindVar(position int) string {
	if m.db.Driver() == repo.DriverPostgres {
		return fmt.Sprintf("$%d", position)
	}

	return "?"
}
