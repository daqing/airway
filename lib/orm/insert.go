package orm

import (
	"context"
	"fmt"
	"strings"
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
	for i, _ := range keys {
		valuePlaceholder = append(valuePlaceholder, fmt.Sprintf("$%d", i))
	}

	sql := "INSERT INTO " + t.TableName() + " (" + strings.Join(keys, ",") + ") VALUES (" + strings.Join(valuePlaceholder, ",") + ") RETURNING *"

	err := db.pool.QueryRow(context.Background(), sql, attributes.Values()...).Scan(&t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// func InsertRecord(db *DB, dst any) error {
// 	if err := db.Create(dst).Error; err != nil {
// 		return err
// 	}

// 	return nil
// }
