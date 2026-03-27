package repo

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
)

func Delete(db *DB, b buildersql.Stmt) error {
	_, err := DeleteAffected(db, b)

	return err
}

func DeleteAffected(db *DB, b buildersql.Stmt) (int64, error) {
	query, args, err := db.prepareBuilder(b)
	if err != nil {
		return 0, err
	}

	result, err := db.conn.ExecContext(context.Background(), query, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, nil
	}

	return rowsAffected, nil
}
