package orm

import (
	"context"
	"fmt"
)

func Delete[T Table](db *DB, cond CondBuilder) error {
	var t T

	where, vals := whereQuery(cond)
	sql := "DELETE FROM " + t.TableName() + " WHERE " + where

	fmt.Println(sql)
	fmt.Println(vals...)

	_, err := db.pool.Exec(context.Background(), sql, vals...)

	return err
}

func DeleteByID[T Table](id any) error {
	return Delete[T](Database(), Eq("id", id))
}
