package repo

import (
	"context"
	"fmt"
	"reflect"

	buildersql "github.com/daqing/airway/lib/sql"
)

func FindByType(db *DB, b buildersql.Stmt, modelType reflect.Type) (any, error) {
	modelType, err := normalizeModelType(modelType)
	if err != nil {
		return nil, err
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

	results := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(modelType)), 0, 0)
	for rows.Next() {
		record := reflect.New(modelType)
		if err := rows.StructScan(record.Interface()); err != nil {
			return nil, err
		}

		results = reflect.Append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results.Interface(), nil
}

func FindOneByType(db *DB, b buildersql.Stmt, modelType reflect.Type) (any, error) {
	rows, err := FindByType(db, b, modelType)
	if err != nil {
		return nil, err
	}

	reflected := reflect.ValueOf(rows)
	if reflected.Len() > 1 {
		return nil, ErrorCountNotMatch
	}

	if reflected.Len() == 0 {
		return nil, nil
	}

	return reflected.Index(0).Interface(), nil
}

func normalizeModelType(modelType reflect.Type) (reflect.Type, error) {
	if modelType == nil {
		return nil, fmt.Errorf("model type cannot be nil")
	}

	for modelType.Kind() == reflect.Pointer {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("model type must be a struct or pointer to struct, got %s", modelType)
	}

	return modelType, nil
}
