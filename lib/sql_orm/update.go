package sql_orm

import (
	"time"

	"github.com/daqing/airway/app/models"
)

func UpdateFields[T TableNameType](id models.IdType, fields *Fields) bool {
	var t T

	db, err := DB()
	if err != nil {
		return false
	}

	now := time.Now().UTC()

	row := fields.ToMap()
	row["updated_at"] = now

	tx := db.Table(t.TableName()).Where("id = ?", id).Updates(row)

	return tx.RowsAffected == 1
}

func UpdateColumn[T TableNameType](cond CondBuilder, field string, value any) bool {
	var t T

	db, err := DB()
	if err != nil {
		return false
	}

	tx := db.Table(t.TableName()).Where(cond.Cond()).Update(field, value)

	return tx.RowsAffected == 1
}
