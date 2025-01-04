package orm

import (
	"fmt"

	"gorm.io/gorm"
)

func FindLike[T Table](db *gorm.DB, fields []string, key string, value string) ([]*T, error) {
	var t T

	likeKey := fmt.Sprintf("%s LIKE ?", key)
	likeValue := fmt.Sprintf("%%%s%%", value)

	tx := db.Table(t.TableName()).Select(fields).Where(likeKey, likeValue)

	var records []*T

	tx.Find(&records)

	return records, nil
}
