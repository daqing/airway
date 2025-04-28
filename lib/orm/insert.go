package orm

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func Insert[T Table](db *DB, attributes *Fields) (*T, error) {
	return InsertSkipExists[T](db, attributes, false)
}

func InsertSkipExists[T Table](db *DB, attributes *Fields, skipExists bool) (*T, error) {
	if skipExists {
		ex, err := Exists[T](db, attributes)
		if err != nil {
			return nil, err
		}

		if ex {
			return nil, nil
		}
	}

	var t T

	keys := attributes.Keys()
	var valuePlaceholder []string
	for i := range len(keys) {
		valuePlaceholder = append(valuePlaceholder, fmt.Sprintf("$%d", i+1))
	}

	sql := "INSERT INTO " + t.TableName() + " (" + strings.Join(keys, ",") + ") VALUES (" + strings.Join(valuePlaceholder, ",") + ") RETURNING *"

	rows, err := db.pool.Query(context.Background(), sql, attributes.Values()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		t, err = pgx.RowToStructByName[T](rows)
		if err != nil {
			return nil, err
		}
	}

	return &t, nil
}

// func InsertRecord(db *DB, dst any) error {
// 	if err := db.Create(dst).Error; err != nil {
// 		return err
// 	}

// 	return nil
// }
