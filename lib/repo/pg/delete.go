package pg

import (
	"context"

	"github.com/daqing/airway/lib/sql"
	"github.com/jackc/pgx/v5"
)

func Delete(db *DB, b *sql.Builder) error {
	sql, vals := b.ToSQL()
	_, err := db.pool.Exec(context.Background(), sql, pgx.NamedArgs(vals))

	return err
}
