package repo

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/daqing/airway/lib/migrate/schema"
)

var typeSpecPattern = regexp.MustCompile(`^([a-zA-Z ]+?)(?:\((\d+)(?:,(\d+))?\))?$`)

type sqliteColumnRow struct {
	CID        int            `db:"cid"`
	Name       string         `db:"name"`
	Type       string         `db:"type"`
	NotNull    int            `db:"notnull"`
	DefaultRaw sql.NullString `db:"dflt_value"`
	PrimaryKey int            `db:"pk"`
}

type sqliteIndexRow struct {
	Seq     int    `db:"seq"`
	Name    string `db:"name"`
	Unique  int    `db:"unique"`
	Origin  string `db:"origin"`
	Partial int    `db:"partial"`
}

type sqliteIndexColumnRow struct {
	Seqno int    `db:"seqno"`
	CID   int    `db:"cid"`
	Name  string `db:"name"`
}

type sqliteForeignKeyRow struct {
	ID       int            `db:"id"`
	Seq      int            `db:"seq"`
	RefTable string         `db:"table"`
	From     string         `db:"from"`
	To       sql.NullString `db:"to"`
	Match    string         `db:"match"`
	OnUpdate string         `db:"on_update"`
	OnDelete string         `db:"on_delete"`
}

type mysqlColumnRow struct {
	Name       string         `db:"column_name"`
	DataType   string         `db:"data_type"`
	ColumnType string         `db:"column_type"`
	IsNullable string         `db:"is_nullable"`
	DefaultRaw sql.NullString `db:"column_default"`
	ColumnKey  string         `db:"column_key"`
	Extra      string         `db:"extra"`
}

type pgColumnRow struct {
	Name             string         `db:"column_name"`
	DataType         string         `db:"data_type"`
	UDTName          string         `db:"udt_name"`
	CharacterMaximum int            `db:"character_maximum_length"`
	NumericPrecision int            `db:"numeric_precision"`
	NumericScale     int            `db:"numeric_scale"`
	IsNullable       string         `db:"is_nullable"`
	DefaultRaw       sql.NullString `db:"column_default"`
}

type indexRow struct {
	Name       string `db:"index_name"`
	ColumnName string `db:"column_name"`
	IsUnique   bool   `db:"is_unique"`
	IsPrimary  bool   `db:"is_primary"`
	SeqInIndex int    `db:"seq_in_index"`
}

type foreignKeyRow struct {
	Name          string         `db:"constraint_name"`
	ColumnName    string         `db:"column_name"`
	RefTable      string         `db:"referenced_table_name"`
	RefColumn     sql.NullString `db:"referenced_column_name"`
	UpdateRule    sql.NullString `db:"update_rule"`
	DeleteRule    sql.NullString `db:"delete_rule"`
	PositionInKey int            `db:"position_in_key"`
}

func InspectSchema(db *DB) (*schema.State, error) {
	tables, err := ListTables(db)
	if err != nil {
		return nil, err
	}

	state := schema.NewState()
	var lastErr error
	for _, tableName := range tables {
		table, err := inspectTableSchema(db, tableName)
		if err != nil {
			lastErr = err
			continue
		}
		state.Tables[tableName] = table
	}

	if len(state.Tables) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return state, nil
}

func inspectTableSchema(db *DB, tableName string) (*schema.TableState, error) {
	switch db.Driver() {
	case DriverSQLite:
		return inspectSQLiteTable(db, tableName)
	case DriverMySQL:
		return inspectMySQLTable(db, tableName)
	default:
		return inspectPostgresTable(db, tableName)
	}
}

func inspectSQLiteTable(db *DB, tableName string) (*schema.TableState, error) {
	ctx := context.Background()
	table := &schema.TableState{Name: tableName}

	createSQL := ""
	if err := db.Conn().GetContext(
		ctx,
		&createSQL,
		db.Conn().Rebind(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`),
		tableName,
	); err != nil {
		return nil, err
	}

	columnRows := []sqliteColumnRow{}
	if err := db.Conn().SelectContext(ctx, &columnRows, fmt.Sprintf("PRAGMA table_info(%s)", quoteSQLiteIdentifier(tableName))); err != nil {
		return nil, err
	}

	table.Columns = make([]schema.Column, 0, len(columnRows))
	for _, row := range columnRows {
		nullable := row.NotNull == 0 && row.PrimaryKey == 0
		column := schema.Column{
			Name:          strings.TrimSpace(row.Name),
			Type:          schemaTypeFromSQLite(row.Type, row.PrimaryKey > 0),
			Null:          schema.Bool(nullable),
			Default:       parseDefaultValue(row.DefaultRaw),
			PrimaryKey:    row.PrimaryKey > 0,
			AutoIncrement: sqliteColumnAutoIncrement(createSQL, row.Name, row.PrimaryKey > 0),
		}
		if column.PrimaryKey {
			column.Null = schema.Bool(false)
		}
		table.Columns = append(table.Columns, column)
	}

	indexRows := []sqliteIndexRow{}
	if err := db.Conn().SelectContext(ctx, &indexRows, fmt.Sprintf("PRAGMA index_list(%s)", quoteSQLiteIdentifier(tableName))); err != nil {
		return nil, err
	}

	for _, row := range indexRows {
		if strings.EqualFold(strings.TrimSpace(row.Origin), "pk") {
			continue
		}

		indexColumns := []sqliteIndexColumnRow{}
		if err := db.Conn().SelectContext(ctx, &indexColumns, fmt.Sprintf("PRAGMA index_info(%s)", quoteSQLiteIdentifier(row.Name))); err != nil {
			return nil, err
		}

		columns := make([]string, 0, len(indexColumns))
		for _, indexColumn := range indexColumns {
			if strings.TrimSpace(indexColumn.Name) == "" {
				continue
			}
			columns = append(columns, indexColumn.Name)
		}
		if len(columns) == 0 {
			continue
		}

		table.Indexes = append(table.Indexes, schema.Index{
			Name:    row.Name,
			Columns: columns,
			Unique:  row.Unique == 1,
		})
	}

	foreignKeyRows := []sqliteForeignKeyRow{}
	if err := db.Conn().SelectContext(ctx, &foreignKeyRows, fmt.Sprintf("PRAGMA foreign_key_list(%s)", quoteSQLiteIdentifier(tableName))); err != nil {
		return nil, err
	}

	for _, row := range foreignKeyRows {
		table.ForeignKeys = append(table.ForeignKeys, schema.ForeignKey{
			Column:    row.From,
			RefTable:  row.RefTable,
			RefColumn: sqlNullStringValue(row.To, "id"),
			OnDelete:  sqlNullStringValue(sql.NullString{String: row.OnDelete, Valid: strings.TrimSpace(row.OnDelete) != ""}, ""),
			OnUpdate:  sqlNullStringValue(sql.NullString{String: row.OnUpdate, Valid: strings.TrimSpace(row.OnUpdate) != ""}, ""),
		})
	}

	markUniqueColumns(table)
	return table, nil
}

func inspectMySQLTable(db *DB, tableName string) (*schema.TableState, error) {
	ctx := context.Background()
	table := &schema.TableState{Name: tableName}

	columnRows := []mysqlColumnRow{}
	columnQuery := db.Conn().Rebind(`SELECT COLUMN_NAME AS column_name, DATA_TYPE AS data_type, COLUMN_TYPE AS column_type, IS_NULLABLE AS is_nullable, COLUMN_DEFAULT AS column_default, COLUMN_KEY AS column_key, EXTRA AS extra
FROM information_schema.columns
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = ?
ORDER BY ORDINAL_POSITION`)
	if err := db.Conn().SelectContext(ctx, &columnRows, columnQuery, tableName); err != nil {
		return nil, err
	}

	table.Columns = make([]schema.Column, 0, len(columnRows))
	for _, row := range columnRows {
		primaryKey := strings.EqualFold(strings.TrimSpace(row.ColumnKey), "PRI")
		autoIncrement := strings.Contains(strings.ToLower(row.Extra), "auto_increment")
		table.Columns = append(table.Columns, schema.Column{
			Name:          row.Name,
			Type:          schemaTypeFromMySQL(row.DataType, row.ColumnType, primaryKey, autoIncrement),
			Null:          schema.Bool(strings.EqualFold(strings.TrimSpace(row.IsNullable), "YES")),
			Default:       parseDefaultValue(row.DefaultRaw),
			PrimaryKey:    primaryKey,
			AutoIncrement: autoIncrement,
			Unique:        strings.EqualFold(strings.TrimSpace(row.ColumnKey), "UNI"),
		})
	}

	indexes, primaryColumns, err := inspectInformationSchemaIndexes(db, tableName)
	if err != nil {
		return nil, err
	}
	table.Indexes = indexes
	markPrimaryColumns(table, primaryColumns)

	foreignKeys, err := inspectMySQLForeignKeys(db, tableName)
	if err != nil {
		return nil, err
	}
	table.ForeignKeys = foreignKeys

	markUniqueColumns(table)
	return table, nil
}

func inspectPostgresTable(db *DB, tableName string) (*schema.TableState, error) {
	ctx := context.Background()
	table := &schema.TableState{Name: tableName}

	columnRows := []pgColumnRow{}
	columnQuery := db.Conn().Rebind(`SELECT column_name,
       data_type,
       udt_name,
       COALESCE(character_maximum_length, 0) AS character_maximum_length,
       COALESCE(numeric_precision, 0) AS numeric_precision,
       COALESCE(numeric_scale, 0) AS numeric_scale,
       is_nullable,
       column_default
FROM information_schema.columns
WHERE table_schema = CURRENT_SCHEMA()
  AND table_name = ?
ORDER BY ordinal_position`)
	if err := db.Conn().SelectContext(ctx, &columnRows, columnQuery, tableName); err != nil {
		return nil, err
	}

	table.Columns = make([]schema.Column, 0, len(columnRows))
	for _, row := range columnRows {
		autoIncrement := strings.Contains(strings.ToLower(row.DefaultRaw.String), "nextval(")
		table.Columns = append(table.Columns, schema.Column{
			Name:          row.Name,
			Type:          schemaTypeFromPostgres(row, false),
			Null:          schema.Bool(strings.EqualFold(strings.TrimSpace(row.IsNullable), "YES")),
			Default:       parseDefaultValue(row.DefaultRaw),
			AutoIncrement: autoIncrement,
		})
	}

	indexes, primaryColumns, err := inspectPostgresIndexes(db, tableName)
	if err != nil {
		return nil, err
	}
	table.Indexes = indexes
	markPrimaryColumns(table, primaryColumns)
	for i := range table.Columns {
		table.Columns[i].Type = schemaTypeFromPostgres(columnRows[i], table.Columns[i].PrimaryKey)
	}

	foreignKeys, err := inspectPostgresForeignKeys(db, tableName)
	if err != nil {
		return nil, err
	}
	table.ForeignKeys = foreignKeys

	markUniqueColumns(table)
	return table, nil
}

func inspectInformationSchemaIndexes(db *DB, tableName string) ([]schema.Index, map[string]bool, error) {
	ctx := context.Background()
	rows := []indexRow{}
	query := db.Conn().Rebind(`SELECT INDEX_NAME AS index_name,
       COLUMN_NAME AS column_name,
       (NON_UNIQUE = 0) AS is_unique,
       (INDEX_NAME = 'PRIMARY') AS is_primary,
       SEQ_IN_INDEX AS seq_in_index
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = ?
ORDER BY INDEX_NAME, SEQ_IN_INDEX`)
	if err := db.Conn().SelectContext(ctx, &rows, query, tableName); err != nil {
		return nil, nil, err
	}

	return buildIndexes(rows), buildPrimaryColumnSet(rows), nil
}

func inspectPostgresIndexes(db *DB, tableName string) ([]schema.Index, map[string]bool, error) {
	ctx := context.Background()
	rows := []indexRow{}
	query := db.Conn().Rebind(`SELECT idx.index_name,
       idx.column_name,
       idx.is_unique,
       idx.is_primary,
       idx.seq_in_index
FROM (
  SELECT i.relname AS index_name,
         a.attname AS column_name,
         ix.indisunique AS is_unique,
         ix.indisprimary AS is_primary,
         ord.ordinality AS seq_in_index
  FROM pg_class t
  JOIN pg_namespace n ON n.oid = t.relnamespace
  JOIN pg_index ix ON ix.indrelid = t.oid
  JOIN pg_class i ON i.oid = ix.indexrelid
  JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS ord(attnum, ordinality) ON TRUE
  JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ord.attnum
  WHERE n.nspname = CURRENT_SCHEMA()
    AND t.relkind = 'r'
    AND t.relname = ?
) AS idx
ORDER BY idx.index_name, idx.seq_in_index`)
	if err := db.Conn().SelectContext(ctx, &rows, query, tableName); err != nil {
		return nil, nil, err
	}

	return buildIndexes(rows), buildPrimaryColumnSet(rows), nil
}

func inspectMySQLForeignKeys(db *DB, tableName string) ([]schema.ForeignKey, error) {
	ctx := context.Background()
	rows := []foreignKeyRow{}
	query := db.Conn().Rebind(`SELECT kcu.CONSTRAINT_NAME AS constraint_name,
       kcu.COLUMN_NAME AS column_name,
       kcu.REFERENCED_TABLE_NAME AS referenced_table_name,
       kcu.REFERENCED_COLUMN_NAME AS referenced_column_name,
       rc.UPDATE_RULE AS update_rule,
       rc.DELETE_RULE AS delete_rule,
       kcu.ORDINAL_POSITION AS position_in_key
FROM information_schema.KEY_COLUMN_USAGE kcu
JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
  ON rc.CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA
 AND rc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
WHERE kcu.TABLE_SCHEMA = DATABASE()
  AND kcu.TABLE_NAME = ?
  AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
ORDER BY kcu.CONSTRAINT_NAME, kcu.ORDINAL_POSITION`)
	if err := db.Conn().SelectContext(ctx, &rows, query, tableName); err != nil {
		return nil, err
	}

	return buildForeignKeys(rows), nil
}

func inspectPostgresForeignKeys(db *DB, tableName string) ([]schema.ForeignKey, error) {
	ctx := context.Background()
	rows := []foreignKeyRow{}
	query := db.Conn().Rebind(`SELECT tc.constraint_name,
       kcu.column_name,
       ccu.table_name AS referenced_table_name,
       ccu.column_name AS referenced_column_name,
       rc.update_rule,
       rc.delete_rule,
       kcu.ordinal_position AS position_in_key
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name
 AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage ccu
  ON ccu.constraint_name = tc.constraint_name
 AND ccu.table_schema = tc.table_schema
JOIN information_schema.referential_constraints rc
  ON rc.constraint_name = tc.constraint_name
 AND rc.constraint_schema = tc.table_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema = CURRENT_SCHEMA()
  AND tc.table_name = ?
ORDER BY tc.constraint_name, kcu.ordinal_position`)
	if err := db.Conn().SelectContext(ctx, &rows, query, tableName); err != nil {
		return nil, err
	}

	return buildForeignKeys(rows), nil
}

func buildIndexes(rows []indexRow) []schema.Index {
	indexes := make([]schema.Index, 0)
	indexByName := map[string]int{}
	for _, row := range rows {
		if row.IsPrimary || strings.TrimSpace(row.ColumnName) == "" {
			continue
		}

		index, ok := indexByName[row.Name]
		if !ok {
			indexByName[row.Name] = len(indexes)
			indexes = append(indexes, schema.Index{Name: row.Name, Unique: row.IsUnique})
			index = len(indexes) - 1
		}
		indexes[index].Columns = append(indexes[index].Columns, row.ColumnName)
	}
	return indexes
}

func buildPrimaryColumnSet(rows []indexRow) map[string]bool {
	primaryColumns := map[string]bool{}
	for _, row := range rows {
		if row.IsPrimary && strings.TrimSpace(row.ColumnName) != "" {
			primaryColumns[row.ColumnName] = true
		}
	}
	return primaryColumns
}

func buildForeignKeys(rows []foreignKeyRow) []schema.ForeignKey {
	foreignKeys := make([]schema.ForeignKey, 0, len(rows))
	for _, row := range rows {
		foreignKeys = append(foreignKeys, schema.ForeignKey{
			Name:      row.Name,
			Column:    row.ColumnName,
			RefTable:  row.RefTable,
			RefColumn: sqlNullStringValue(row.RefColumn, "id"),
			OnDelete:  sqlNullStringValue(row.DeleteRule, ""),
			OnUpdate:  sqlNullStringValue(row.UpdateRule, ""),
		})
	}
	return foreignKeys
}

func markPrimaryColumns(table *schema.TableState, primaryColumns map[string]bool) {
	for i := range table.Columns {
		if !primaryColumns[table.Columns[i].Name] {
			continue
		}
		table.Columns[i].PrimaryKey = true
		table.Columns[i].Null = schema.Bool(false)
		if table.Columns[i].Type.Kind == schema.TypeInteger || table.Columns[i].Type.Kind == schema.TypeBigInt {
			table.Columns[i].Type.Kind = schema.TypeID
		}
	}
}

func markUniqueColumns(table *schema.TableState) {
	for _, index := range table.Indexes {
		if !index.Unique || len(index.Columns) != 1 {
			continue
		}

		columnName := index.Columns[0]
		for i := range table.Columns {
			if table.Columns[i].Name == columnName {
				table.Columns[i].Unique = true
			}
		}
	}
}

func schemaTypeFromSQLite(typeDecl string, primaryKey bool) schema.Type {
	base, length, precision, scale := parseTypeSpec(typeDecl)
	kind := strings.ToLower(base)

	switch {
	case strings.Contains(kind, "json"):
		return schema.Type{Kind: schema.TypeJSON}
	case strings.Contains(kind, "timestamp"), strings.Contains(kind, "datetime"), kind == "date", kind == "time":
		return schema.Type{Kind: schema.TypeDateTime}
	case strings.Contains(kind, "bool"):
		return schema.Type{Kind: schema.TypeBoolean}
	case strings.Contains(kind, "bigint"):
		if primaryKey {
			return schema.Type{Kind: schema.TypeID}
		}
		return schema.Type{Kind: schema.TypeBigInt}
	case strings.Contains(kind, "int"):
		if primaryKey {
			return schema.Type{Kind: schema.TypeID}
		}
		return schema.Type{Kind: schema.TypeInteger}
	case strings.Contains(kind, "char"), strings.Contains(kind, "clob"):
		return schema.Type{Kind: schema.TypeString, Length: length}
	case strings.Contains(kind, "text"):
		return schema.Type{Kind: schema.TypeText}
	default:
		if length > 0 {
			return schema.Type{Kind: schema.TypeString, Length: length}
		}
		if precision > 0 || scale > 0 {
			return schema.Type{Kind: schema.TypeString, Precision: precision, Scale: scale}
		}
		return schema.Type{Kind: schema.TypeString}
	}
}

func schemaTypeFromMySQL(dataType string, columnType string, primaryKey bool, autoIncrement bool) schema.Type {
	base, length, precision, scale := parseTypeSpec(columnType)
	kind := strings.ToLower(strings.TrimSpace(dataType))
	if kind == "" {
		kind = strings.ToLower(base)
	}

	switch {
	case kind == "json":
		return schema.Type{Kind: schema.TypeJSON}
	case kind == "datetime", kind == "timestamp", kind == "date":
		return schema.Type{Kind: schema.TypeDateTime}
	case kind == "tinyint" && strings.HasPrefix(strings.ToLower(strings.TrimSpace(columnType)), "tinyint(1)"):
		return schema.Type{Kind: schema.TypeBoolean}
	case strings.Contains(kind, "bool"):
		return schema.Type{Kind: schema.TypeBoolean}
	case kind == "bigint":
		if primaryKey || autoIncrement {
			return schema.Type{Kind: schema.TypeID}
		}
		return schema.Type{Kind: schema.TypeBigInt}
	case kind == "int" || kind == "integer" || kind == "smallint" || kind == "mediumint" || kind == "tinyint":
		if primaryKey || autoIncrement {
			return schema.Type{Kind: schema.TypeID}
		}
		return schema.Type{Kind: schema.TypeInteger}
	case kind == "char" || kind == "varchar":
		return schema.Type{Kind: schema.TypeString, Length: length}
	case strings.Contains(kind, "text"):
		return schema.Type{Kind: schema.TypeText}
	default:
		if precision > 0 || scale > 0 {
			return schema.Type{Kind: schema.TypeString, Precision: precision, Scale: scale}
		}
		return schema.Type{Kind: schema.TypeString}
	}
}

func schemaTypeFromPostgres(row pgColumnRow, primaryKey bool) schema.Type {
	kind := strings.ToLower(strings.TrimSpace(row.DataType))
	udtName := strings.ToLower(strings.TrimSpace(row.UDTName))

	switch {
	case kind == "json" || kind == "jsonb":
		return schema.Type{Kind: schema.TypeJSON}
	case strings.Contains(kind, "timestamp"), kind == "date", kind == "time without time zone", kind == "time with time zone":
		return schema.Type{Kind: schema.TypeDateTime}
	case kind == "boolean":
		return schema.Type{Kind: schema.TypeBoolean}
	case udtName == "int8" || kind == "bigint":
		if primaryKey {
			return schema.Type{Kind: schema.TypeID}
		}
		return schema.Type{Kind: schema.TypeBigInt}
	case udtName == "int2" || udtName == "int4" || kind == "integer" || kind == "smallint":
		if primaryKey {
			return schema.Type{Kind: schema.TypeID}
		}
		return schema.Type{Kind: schema.TypeInteger}
	case kind == "character varying" || kind == "character":
		return schema.Type{Kind: schema.TypeString, Length: row.CharacterMaximum}
	case kind == "text":
		return schema.Type{Kind: schema.TypeText}
	default:
		if row.CharacterMaximum > 0 {
			return schema.Type{Kind: schema.TypeString, Length: row.CharacterMaximum}
		}
		if row.NumericPrecision > 0 || row.NumericScale > 0 {
			return schema.Type{Kind: schema.TypeString, Precision: row.NumericPrecision, Scale: row.NumericScale}
		}
		return schema.Type{Kind: schema.TypeString}
	}
}

func parseTypeSpec(typeDecl string) (string, int, int, int) {
	matches := typeSpecPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(typeDecl)))
	if len(matches) == 0 {
		return strings.TrimSpace(typeDecl), 0, 0, 0
	}

	first, _ := strconv.Atoi(matches[2])
	second, _ := strconv.Atoi(matches[3])
	return strings.TrimSpace(matches[1]), first, first, second
}

func parseDefaultValue(raw sql.NullString) any {
	if !raw.Valid {
		return nil
	}

	value := strings.TrimSpace(raw.String)
	if value == "" || strings.EqualFold(value, "null") {
		return nil
	}

	for strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		value = strings.TrimSpace(value[1 : len(value)-1])
	}

	if len(value) >= 2 {
		if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
			return value[1 : len(value)-1]
		}
	}

	if strings.EqualFold(value, "true") {
		return true
	}
	if strings.EqualFold(value, "false") {
		return false
	}

	if integerValue, err := strconv.ParseInt(value, 10, 64); err == nil {
		return integerValue
	}

	if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
		return floatValue
	}

	return value
}

func sqliteColumnAutoIncrement(createSQL string, columnName string, primaryKey bool) bool {
	if !primaryKey {
		return false
	}

	normalized := strings.ToLower(createSQL)
	columnName = strings.ToLower(strings.TrimSpace(columnName))
	return strings.Contains(normalized, columnName) && strings.Contains(normalized, "autoincrement")
}

func quoteSQLiteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(strings.TrimSpace(value), `"`, `""`) + `"`
}

func sqlNullStringValue(value sql.NullString, fallback string) string {
	if !value.Valid {
		return fallback
	}
	return strings.TrimSpace(value.String)
}
