package pg

import (
	"context"

	"github.com/jmoiron/sqlx"
)

func Tx(db *DB, fn func(tx *sqlx.Tx) error) error {
	ctx := context.Background()

	tx, err := db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = fn(tx)
	if err != nil {
		return err
	}

	return tx.Commit()
}
