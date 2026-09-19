package repo

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
)

func Count(db *DB, b buildersql.Stmt) (n int64, err error) {
	return CountWith(db.executor(), b)
}

func CountWith(ex *Executor, b buildersql.Stmt) (n int64, err error) {
	query, args, err := ex.prepareBuilder(b)
	if err != nil {
		return 0, err
	}

	err = ex.q.QueryRowContext(context.Background(), query, args...).Scan(&n)

	return n, err
}
