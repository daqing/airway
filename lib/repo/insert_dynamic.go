package repo

import (
	"context"
	"reflect"

	buildersql "github.com/daqing/airway/lib/sql"
)

func InsertByType(db *DB, b buildersql.Stmt, modelType reflect.Type) (any, error) {
	return InsertByTypeWith(db.executor(), b, modelType)
}

func InsertByTypeWith(ex *Executor, b buildersql.Stmt, modelType reflect.Type) (any, error) {
	modelType, err := normalizeModelType(modelType)
	if err != nil {
		return nil, err
	}

	if ex.driver == DriverMySQL {
		return insertMySQLByType(ex, b, modelType)
	}

	query, args, err := ex.prepareBuilder(b)
	if err != nil {
		return nil, err
	}

	rows, err := ex.q.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	record := reflect.New(modelType)
	rowScanner := newStructScanner(rows)
	for rows.Next() {
		if err := rowScanner.Scan(record.Interface()); err != nil {
			return nil, err
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return record.Interface(), nil
}

func insertMySQLByType(ex *Executor, b buildersql.Stmt, modelType reflect.Type) (any, error) {
	query, args, err := ex.prepareInsertBuilder(b)
	if err != nil {
		return nil, err
	}

	result, err := ex.q.ExecContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}

	lookupColumn, lookupValue, err := resolveInsertLookup(ex, b, result)
	if err != nil {
		return nil, err
	}

	selectQuery := "SELECT * FROM " + b.TableName() + " WHERE " + lookupColumn + " = @lookup LIMIT 1"
	compiledQuery, compiledArgs, err := ex.prepareQuery(selectQuery, buildersql.NamedArgs{"lookup": lookupValue})
	if err != nil {
		return nil, err
	}

	record := reflect.New(modelType)
	if err := getStruct(context.Background(), ex.q, record.Interface(), compiledQuery, compiledArgs...); err != nil {
		return nil, err
	}

	return record.Interface(), nil
}
