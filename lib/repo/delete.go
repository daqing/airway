package repo

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
)

func Delete(db *DB, b buildersql.Stmt) error {
	query, args, err := db.prepareBuilder(b)
	if err != nil {
		return err
	}

	_, err = db.conn.ExecContext(context.Background(), query, args...)

	return err
}
