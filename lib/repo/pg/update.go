package pg

import (
	"context"

	buildersql "github.com/daqing/airway/lib/sql"
)

func Update(db *DB, b *buildersql.Builder) error {
	query, args, err := db.prepareBuilder(b)
	if err != nil {
		return err
	}

	_, err = db.conn.ExecContext(context.Background(), query, args...)

	return err
}
