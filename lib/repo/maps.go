package repo

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
	"github.com/jmoiron/sqlx"
)

func Preview(db *DB, b buildersql.Stmt) (string, []any, error) {
	return db.prepareBuilder(b)
}

func FindOneMap(db *DB, b buildersql.Stmt) (map[string]any, error) {
	rows, err := FindMaps(db, b)
	if err != nil {
		return nil, err
	}

	if len(rows) > 1 {
		return nil, ErrorCountNotMatch
	}

	if len(rows) == 0 {
		return nil, nil
	}

	return rows[0], nil
}

func FindMaps(db *DB, b buildersql.Stmt) ([]map[string]any, error) {
	query, args, err := db.prepareBuilder(b)
	if err != nil {
		return nil, err
	}

	rows, err := db.conn.QueryxContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return scanMapRows(rows)
}

func InsertMap(db *DB, b buildersql.Stmt) (map[string]any, error) {
	if db.Driver() == DriverMySQL {
		return insertMySQLMap(db, b)
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

	records, err := scanMapRows(rows)
	if err != nil {
		return nil, err
	}

	if len(records) > 1 {
		return nil, ErrorCountNotMatch
	}

	if len(records) == 0 {
		return nil, nil
	}

	return records[0], nil
}

func insertMySQLMap(db *DB, b buildersql.Stmt) (map[string]any, error) {
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

	selectQuery := "SELECT * FROM " + b.TableName() + " WHERE " + lookupColumn + " = @lookup LIMIT 1"
	compiledQuery, compiledArgs, err := db.prepareQuery(selectQuery, buildersql.NamedArgs{"lookup": lookupValue})
	if err != nil {
		return nil, err
	}

	rows, err := db.conn.QueryxContext(context.Background(), compiledQuery, compiledArgs...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	records, err := scanMapRows(rows)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	return records[0], nil
}

func scanMapRows(rows *sqlx.Rows) ([]map[string]any, error) {
	records := make([]map[string]any, 0)

	for rows.Next() {
		record := map[string]any{}
		if err := rows.MapScan(record); err != nil {
			return nil, err
		}

		records = append(records, normalizeRecord(record))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func normalizeRecord(record map[string]any) map[string]any {
	normalized := make(map[string]any, len(record))
	for key, value := range record {
		normalized[key] = normalizeRecordValue(value)
	}

	return normalized
}

func normalizeRecordValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	case map[string]any:
		return normalizeRecord(typed)
	case []any:
		normalized := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized = append(normalized, normalizeRecordValue(item))
		}

		return normalized
	default:
		return value
	}
}
