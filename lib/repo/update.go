package repo

import (
	"context"
	"fmt"
	"log"
)

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
