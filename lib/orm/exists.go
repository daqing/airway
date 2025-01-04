package orm

import "gorm.io/gorm"

func Exists[T Table](db *gorm.DB, cond CondBuilder) (bool, error) {
	n, err := Count[T](db, cond)

	return n > 0, err
}
