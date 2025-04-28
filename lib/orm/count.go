package orm

import (
	"context"
	"fmt"
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

	i := 0
	for k, v := range cond.Cond() {
		i++
		condStr = append(condStr, fmt.Sprintf("%s = $%d", k, i))
		result = append(result, v)
	}

	return strings.Join(condStr, " AND "), result
}
