package orm

import (
	"context"
	"fmt"
	"strings"
)

func FindLike[T Table](db *DB, fields []string, key string, value string) ([]*T, error) {
	var t T

	likeKey := fmt.Sprintf("%s LIKE ?", key)
	likeValue := fmt.Sprintf("%%%s%%", value)

	sql := "SELECT " + strings.Join(fields, ",") + " FROM " + t.TableName() + " WHERE " + likeKey

	var records []*T
	rows, err := db.pool.Query(context.Background(), sql, likeValue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var record T
		err := rows.Scan(&record)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}
