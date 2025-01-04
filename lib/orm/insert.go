package orm

import (
	"time"

	"gorm.io/gorm"
)

func Insert[T Table](db *gorm.DB, attributes *Fields) (*T, error) {
	return InsertSkipExists[T](db, attributes, false)
}

func InsertSkipExists[T Table](db *gorm.DB, attributes *Fields, skipExists bool) (*T, error) {
	if skipExists {
		ex, err := Exists[T](db, attributes)
		if err != nil {
			return nil, err
		}

		if ex {
			return nil, nil
		}
	}

	var t T

	row := attributes.ToMap()

	now := time.Now().UTC()
	row["created_at"] = now
	row["updated_at"] = now

	if err := db.Table(t.TableName()).Create(row).Scan(&t).Error; err != nil {
		return nil, err
	}

	return &t, nil
}

func InsertRecord(db *gorm.DB, dst any) error {
	if err := db.Create(dst).Error; err != nil {
		return err
	}

	return nil
}
