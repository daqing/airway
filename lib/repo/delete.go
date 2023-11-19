package repo

import (
	"context"
	"fmt"
)

func Delete[T TableNameType](conds []KVPair) error {
	var t T

	condQuery, vals, _ := buildCondQuery(conds, 0, and_sep)

	sql := fmt.Sprintf("DELETE FROM %s WHERE %s", t.TableName(), condQuery)
	fmt.Println(sql)

	_, err := Pool.Exec(context.Background(), sql, vals...)
	if err != nil {
		return err
	}

	return nil
}
