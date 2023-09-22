package repo

import (
	"context"
	"fmt"
	"log"
)

func UpdateRow[T TableNameType, Id int | int64](id Id, field string, value any) bool {
	fmt.Printf("id=%d, field=%s, value=%s\n", id, field, value)

	var t T

	sql := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE id = $2", t.TableName(), field)

	row, err := Pool.Exec(context.Background(), sql, value, id)
	if err != nil {
		log.Printf("conn.Exec error: %s\n", err.Error())
		return false
	}

	log.Printf("affected rows: %d\n", row.RowsAffected())

	return row.RowsAffected() == 1
}
