package repo

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
)

func FindOne[T any](db *DB, b buildersql.Stmt) (*T, error) {
	return FindOneWith[T](db.executor(), b)
}

func FindOneWith[T any](ex *Executor, b buildersql.Stmt) (*T, error) {
	rows, err := FindWith[T](ex, b)
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

// limit = 0 means no limit
func Find[T any](db *DB, b buildersql.Stmt) ([]*T, error) {
	return FindWith[T](db.executor(), b)
}

func FindWith[T any](ex *Executor, b buildersql.Stmt) ([]*T, error) {
	var records = []*T{}

	query, args, err := ex.prepareBuilder(b)
	if err != nil {
		return nil, err
	}

	rows, err := ex.q.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	rowScanner := newStructScanner(rows)
	for rows.Next() {
		var record T
		if err := rowScanner.Scan(&record); err != nil {
			return nil, err
		}

		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}
