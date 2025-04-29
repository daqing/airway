package pg_repo

import (
	"context"

	"github.com/daqing/airway/lib/sql"
	"github.com/jackc/pgx/v5"
)

func FindOne[T any](db *DB, b *sql.Builder) (*T, error) {
	rows, err := Find[T](db, b)
	if err != nil {
		return nil, err
	}

	if len(rows) > 1 {
		return nil, ErrorCountNotMatch
	}

	if len(rows) == 0 {
		return nil, ErrorNotFound
	}

	return rows[0], nil
}

// limit = 0 means no limit
func Find[T any](db *DB, b *sql.Builder) ([]*T, error) {
	var records = []*T{}

	sql, vals := b.ToSQL()
	rows, err := db.pool.Query(context.Background(), sql, pgx.NamedArgs(vals))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var record T
		record, err := pgx.RowToStructByName[T](rows)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}
