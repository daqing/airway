package repo

import (
	"context"
	"fmt"
	"log"
	"strings"
)

func UpdateFields[T TableNameType](id int64, fields []KeyValueField) bool {
	var t T

	n := 0
	var keys []string
	var vals []any

	for _, kv := range fields {
		n++
		keys = append(keys, fmt.Sprintf("%s = $%d", kv.KeyField(), n))
		vals = append(vals, kv.ValueField())
	}

	fieldQuery := strings.Join(keys, ",")

	n++
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", t.TableName(), fieldQuery, n)

	vals = append(vals, id)

	row, err := Pool.Exec(context.Background(), sql, vals...)

	if err != nil {
		log.Printf("conn.Exec error: %s\n", err.Error())
		return false
	}

	return row.RowsAffected() == 1
}

func UpdateRow[T TableNameType](id int64, field string, value any) bool {
	var t T

	sql := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE id = $2", t.TableName(), field)

	row, err := Pool.Exec(context.Background(), sql, value, id)
	if err != nil {
		log.Printf("conn.Exec error: %s\n", err.Error())
		return false
	}

	return row.RowsAffected() == 1
}
