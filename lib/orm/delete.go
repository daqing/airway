package orm

import (
	"gorm.io/gorm"
)

func Delete[T TableNameType](db *gorm.DB, cond CondBuilder) error {
	var t T

	db.Table(t.TableName()).Where(cond.Cond()).Delete(&t)

	return nil
}

func DeleteByID[T TableNameType](id any) error {
	return Delete[T](DB(), Eq("id", id))
}
