package repo

import (
	"context"
	"fmt"
	"log"
)

func UpdateFields[T TableNameType](id int64, fields []KVPair) bool {
	var t T

	condQuery, vals, n := buildCondQuery(fields, 0, comma_sep)

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", t.TableName(), condQuery, n)

	vals = append(vals, id)

	row, err := Pool.Exec(context.Background(), sql, vals...)

	if err != nil {
		log.Printf("conn.Exec error: %s\n", err.Error())
		return false
	}

	return row.RowsAffected() == 1
}

func UpdateRow[T TableNameType](cond []KVPair, field string, value any) bool {
	var t T

	fieldQuery, vals, _ := buildCondQuery(cond, 1, and_sep)

	sql := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s", t.TableName(), field, fieldQuery)

	// prepend value to be the first $1 value
	vals = append([]any{value}, vals...)

	row, err := Pool.Exec(context.Background(), sql, vals...)
	if err != nil {
		log.Printf("conn.Exec error: %s\n", err.Error())
		return false
	}

	return row.RowsAffected() == 1
}
