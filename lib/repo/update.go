package repo

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
)

func Update(db *DB, b buildersql.Stmt) error {
	return UpdateWith(db.executor(), b)
}

func UpdateWith(ex *Executor, b buildersql.Stmt) error {
	_, err := UpdateAffectedWith(ex, b)

	return err
}

func UpdateAffected(db *DB, b buildersql.Stmt) (int64, error) {
	return UpdateAffectedWith(db.executor(), b)
}

func UpdateAffectedWith(ex *Executor, b buildersql.Stmt) (int64, error) {
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
