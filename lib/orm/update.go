package orm

import (
	"time"

	"github.com/daqing/airway/app/models"
	"gorm.io/gorm"
)

func UpdateFields[T Table](db *gorm.DB, id models.IdType, fields *Fields) bool {
	var t T

	now := time.Now().UTC()

	row := fields.ToMap()
	row["updated_at"] = now

	tx := db.Table(t.TableName()).Where("id = ?", id).Updates(row)

	return tx.RowsAffected == 1
}

func UpdateColumn[T Table](db *gorm.DB, cond CondBuilder, field string, value any) bool {
	var t T

	tx := db.Table(t.TableName()).Where(cond.Cond()).Update(field, value)

	return tx.RowsAffected == 1
}
