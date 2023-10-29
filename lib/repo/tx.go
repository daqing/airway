package repo

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type TxFunc func(tx pgx.Tx, ctx context.Context) error

func Tx(fn TxFunc) error {
	ctx := context.Background()

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return err
	}

	err = fn(tx, ctx)

	if err != nil {
		return tx.Rollback(ctx)
	}

	return tx.Commit(ctx)
}
