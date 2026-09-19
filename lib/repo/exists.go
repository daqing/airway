package repo

import "github.com/daqing/airway/lib/sql"

func Exists(db *DB, b sql.Stmt) (bool, error) {
	return ExistsWith(db.executor(), b)
}

func ExistsWith(ex *Executor, b sql.Stmt) (bool, error) {
	n, err := CountWith(ex, b)

	return n > 0, err
}
