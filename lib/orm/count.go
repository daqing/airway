package orm

import (
	"context"
	"strings"
)

func Count[T Table](db *DB, cond CondBuilder) (n int64, err error) {
	var t T

	where, vals := whereQuery(cond)
	sql := "SELECT COUNT(*) FROM " + t.TableName() + " WHERE " + where
	db.pool.QueryRow(context.Background(), sql, vals...).Scan(&n)

	return n, nil
}

// map[string]any -> where foo = ? and bar = ?, args = [foo, bar]
func whereQuery(cond CondBuilder) (string, []any) {
	var condStr []string
	var result []any
	for k, v := range cond.Cond() {
		condStr = append(condStr, k+" = ?")
		result = append(result, v)
	}

	return strings.Join(condStr, " AND "), result
}
