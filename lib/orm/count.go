package orm

import "gorm.io/gorm"

func Count[T Table](db *gorm.DB, cond CondBuilder) (n int64, err error) {
	var t T

	db.Table(t.TableName()).Where(cond.Cond()).Count(&n)

	return n, nil
}
