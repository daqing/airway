package pg

import "github.com/daqing/airway/lib/sql"

func Exists(db *DB, b *sql.Builder) (bool, error) {
	n, err := Count(db, b)

	return n > 0, err
}
