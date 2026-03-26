package repo

import "github.com/daqing/airway/lib/sql"

func Exists(db *DB, b sql.Stmt) (bool, error) {
	n, err := Count(db, b)

	return n > 0, err
}
