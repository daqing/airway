package repo

import (
	"context"
	"fmt"
	"strings"

	buildersql "github.com/daqing/airway/lib/sql"
)

func Create[T any](db *DB, b buildersql.Stmt) (*T, error) {
	return Insert[T](db, b)
}

func Insert[T any](db *DB, b buildersql.Stmt) (*T, error) {
	return insertSkipExists[T](db, b, false)
}

func insertSkipExists[T any](db *DB, b buildersql.Stmt, skipExists bool) (*T, error) {
	if skipExists {
		ex, err := Exists(db, b)
		if err != nil {
			return nil, err
		}

		if ex {
			return nil, nil
		}
	}

	var t T
	if db.Driver() == DriverMySQL {
		return insertMySQL[T](db, b)
	}

	query, args, err := db.prepareBuilder(b)
	if err != nil {
		return nil, err
	}

	rows, err := db.conn.QueryxContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		if err := rows.StructScan(&t); err != nil {
			return nil, err
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &t, nil
}

func insertMySQL[T any](db *DB, b buildersql.Stmt) (*T, error) {
	query, args, err := db.prepareInsertBuilder(b)
	if err != nil {
		return nil, err
	}

	result, err := db.conn.ExecContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}

	lookupColumn, lookupValue, err := db.resolveInsertLookup(b, result)
	if err != nil {
		return nil, err
	}

	selectQuery := fmt.Sprintf("SELECT * FROM %s WHERE %s = @lookup LIMIT 1", b.TableName(), lookupColumn)
	compiledQuery, compiledArgs, err := db.prepareQuery(selectQuery, buildersql.NamedArgs{"lookup": lookupValue})
	if err != nil {
		return nil, err
	}

	var record T
	if err := db.conn.GetContext(context.Background(), &record, compiledQuery, compiledArgs...); err != nil {
		return nil, err
	}

	return &record, nil
}

func (db *DB) resolveInsertLookup(b buildersql.Stmt, result any) (string, any, error) {
	if execResult, ok := result.(interface{ LastInsertId() (int64, error) }); ok {
		lastInsertID, err := execResult.LastInsertId()
		if err == nil && lastInsertID > 0 {
			primaryKey, pkErr := db.lookupPrimaryKeyColumn(b.TableName())
			if pkErr == nil {
				return primaryKey, lastInsertID, nil
			}
		}
	}

	vals := b.InsertValues()
	if value, ok := lookupInsertValue(vals, "id"); ok {
		return "id", value, nil
	}

	for _, column := range b.ConflictTarget() {
		if value, ok := lookupInsertValue(vals, column); ok {
			return column, value, nil
		}
	}

	rows := b.InsertRows()
	if len(rows) == 1 {
		if value, ok := lookupInsertValue(rows[0], "id"); ok {
			return "id", value, nil
		}

		for _, column := range b.ConflictTarget() {
			if value, ok := lookupInsertValue(rows[0], column); ok {
				return column, value, nil
			}
		}
	}

	return "", nil, fmt.Errorf("mysql insert fallback requires a retrievable primary or conflict key")
}

func lookupInsertValue(vals buildersql.H, column string) (any, bool) {
	if len(vals) == 0 {
		return nil, false
	}

	target := normalizeLookupColumn(column)
	for key, value := range vals {
		if normalizeLookupColumn(key) == target {
			return value, true
		}
	}

	return nil, false
}

func normalizeLookupColumn(column string) string {
	column = strings.TrimSpace(column)
	column = strings.Trim(column, "`\"")
	parts := strings.Split(column, ".")
	if len(parts) == 0 {
		return column
	}

	return strings.Trim(parts[len(parts)-1], "`\"")
}

func (db *DB) lookupPrimaryKeyColumn(tableName string) (string, error) {
	cleanTable := normalizeLookupColumn(tableName)
	if cleanTable == "" {
		return "", fmt.Errorf("table name is empty")
	}

	var primaryKey string
	err := db.conn.GetContext(
		context.Background(),
		&primaryKey,
		`SELECT COLUMN_NAME
FROM information_schema.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME = ?
  AND CONSTRAINT_NAME = 'PRIMARY'
ORDER BY ORDINAL_POSITION
LIMIT 1`,
		cleanTable,
	)
	if err != nil {
		return "", err
	}

	return primaryKey, nil
}
