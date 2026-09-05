package repo

import (
	"context"
	"database/sql"
)

func Tx(db *DB, fn func(tx *sql.Tx) error) error {
	ctx := context.Background()

	tx, err := db.conn.BeginTx(ctx, nil)
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
