package repo

import (
	"context"
	"fmt"
)

func Count[T TableNameType](conds []KVPair) (n int64, err error) {
	condQuery, values, _ := buildCondQuery(conds, 0, and_sep)

	var t T

	sql := fmt.Sprintf("select count(*) from %s WHERE %s", t.TableName(), condQuery)
	row := Pool.QueryRow(context.Background(), sql, values...)

	err = row.Scan(&n)

	return
}
