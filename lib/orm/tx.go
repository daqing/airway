package orm

import "gorm.io/gorm"

func Tx(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
