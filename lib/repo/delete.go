package repo

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
)

func Delete(db *DB, b buildersql.Stmt) error {
	return DeleteWith(db.executor(), b)
}

func DeleteWith(ex *Executor, b buildersql.Stmt) error {
	_, err := DeleteAffectedWith(ex, b)

	return err
}

func DeleteAffected(db *DB, b buildersql.Stmt) (int64, error) {
	return DeleteAffectedWith(db.executor(), b)
}

func DeleteAffectedWith(ex *Executor, b buildersql.Stmt) (int64, error) {
	query, args, err := ex.prepareBuilder(b)
	if err != nil {
		return 0, err
	}

	result, err := ex.q.ExecContext(context.Background(), query, args...)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, nil
	}

	return rowsAffected, nil
}
