package pg

import (
	"context"

	"github.com/daqing/airway/lib/sql"
	"github.com/jackc/pgx/v5"
)

func Insert[T any](db *DB, b *sql.Builder) (*T, error) {
	return insertSkipExists[T](db, b, false)
}

func insertSkipExists[T any](db *DB, b *sql.Builder, skipExists bool) (*T, error) {
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

	sql, vals := b.ToSQL()

	rows, err := db.pool.Query(context.Background(), sql, pgx.NamedArgs(vals))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		t, err = pgx.RowToStructByName[T](rows)
		if err != nil {
			return nil, err
		}
	}

	return &t, nil
}
