package repo

import (
	"context"
	"fmt"
)

func Count[T TableNameType](conds []KeyValueField) (n int64, err error) {
	condQuery, values := buildCondQuery(conds)

	var t T

	sql := fmt.Sprintf("select count(*) from %s WHERE %s", t.TableName(), condQuery)

	fmt.Println("---> Count SQL:", sql, "values:", values)

	row := Pool.QueryRow(context.Background(), sql, values...)

	err = row.Scan(&n)

	fmt.Println("---> Rows affected:", n)

	return
}
