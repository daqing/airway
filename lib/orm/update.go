package orm

import (
	"context"
	"strings"

	"github.com/daqing/airway/app/models"
)

func UpdateFields[T Table](db *DB, id models.IdType, fields *Fields) bool {
	var t T

	sql := "UPDATE " + t.TableName() + " SET "

	var keys []string
	i := 0
	for _, key := range fields.Keys() {
		keys = append(keys, key+" = $"+string(i))
		i++
	}

	sql += strings.Join(keys, ",") + " WHERE id = $" + string(i)

	var args []any = fields.Values()
	args = append(args, id)

	_, err := db.pool.Exec(context.Background(), sql, args...)
	if err != nil {
		return false
	}

	return true
}

// func UpdateColumn[T Table](db *DB, cond CondBuilder, field string, value any) bool {
// 	var t T

// 	tx := db.Table(t.TableName()).Where(cond.Cond()).Update(field, value)

// 	return tx.RowsAffected == 1
// }
