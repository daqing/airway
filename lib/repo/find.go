package repo

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
)

func FindOne[T any](db *DB, b buildersql.Stmt) (*T, error) {
	rows, err := Find[T](db, b)
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
	var records = []*T{}

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
		var record T
		if err := rows.StructScan(&record); err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}
