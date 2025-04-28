package orm

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func Tx(db *DB, fn func(tx pgx.Tx) error) error {
	ctx := context.Background()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = fn(tx)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
