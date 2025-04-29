package pg

import (
	"context"

	"github.com/daqing/airway/lib/sql"
	"github.com/jackc/pgx/v5"
)

func Count(db *DB, b *sql.Builder) (n int64, err error) {
	sql, vals := b.ToSQL()

	db.pool.QueryRow(context.Background(), sql, pgx.NamedArgs(vals)).Scan(&n)

	return n, nil
}
