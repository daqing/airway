package repo

import (
	"context"
	"reflect"

	buildersql "github.com/daqing/airway/lib/sql"
)

func InsertByType(db *DB, b buildersql.Stmt, modelType reflect.Type) (any, error) {
	modelType, err := normalizeModelType(modelType)
	if err != nil {
		return nil, err
	}

	if db.Driver() == DriverMySQL {
		return insertMySQLByType(db, b, modelType)
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

	record := reflect.New(modelType)
	for rows.Next() {
		if err := rows.StructScan(record.Interface()); err != nil {
			return nil, err
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return record.Interface(), nil
}

func insertMySQLByType(db *DB, b buildersql.Stmt, modelType reflect.Type) (any, error) {
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

	record := reflect.New(modelType)
	if err := db.conn.GetContext(context.Background(), record.Interface(), compiledQuery, compiledArgs...); err != nil {
		return nil, err
	}

	return record.Interface(), nil
}
