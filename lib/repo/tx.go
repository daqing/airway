package repo

import (
	"context"
	"database/sql"

	buildersql "github.com/daqing/airway/lib/sql"
)

type Tx struct {
	tx   *sql.Tx
	exec *Executor
}

func WithTx(db *DB, fn func(tx *Tx) error) error {
	return WithTxContext(context.Background(), db, fn)
}

func WithTxContext(ctx context.Context, db *DB, fn func(tx *Tx) error) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(&Tx{tx: tx, exec: &Executor{q: tx, driver: db.driver}}); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) Raw() *sql.Tx {
	return t.tx
}

func (t *Tx) Executor() *Executor {
	return t.exec
}

func (t *Tx) Update(b buildersql.Stmt) error {
	return UpdateWith(t.exec, b)
}

func (t *Tx) UpdateAffected(b buildersql.Stmt) (int64, error) {
	return UpdateAffectedWith(t.exec, b)
}

func (t *Tx) Delete(b buildersql.Stmt) error {
	return DeleteWith(t.exec, b)
}

func (t *Tx) Count(b buildersql.Stmt) (int64, error) {
	return CountWith(t.exec, b)
}

func (t *Tx) Exists(b buildersql.Stmt) (bool, error) {
	return ExistsWith(t.exec, b)
}
