package repo

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
)

func Count(db *DB, b buildersql.Stmt) (n int64, err error) {
	query, args, err := db.prepareBuilder(b)
	if err != nil {
		return 0, err
	}

	err = db.conn.QueryRowxContext(context.Background(), query, args...).Scan(&n)

	return n, err
}
