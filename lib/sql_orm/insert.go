package sql_orm

import "time"

func Insert[T TableNameType](attributes *Fields) (*T, error) {
	return InsertSkipExists[T](attributes, false)
}

func InsertSkipExists[T TableNameType](attributes *Fields, skipExists bool) (*T, error) {
	if skipExists {
		ex, err := Exists[T](attributes)
		if err != nil {
			return nil, err
		}

		if ex {
			return nil, nil
		}
	}

	var t T

	db, err := DB()
	if err != nil {
		return nil, err
	}

	row := attributes.ToMap()

	now := time.Now().UTC()
	row["created_at"] = now
	row["updated_at"] = now

	if err := db.Table(t.TableName()).Create(row).Scan(&t).Error; err != nil {
		return nil, err
	}

	return &t, nil
}

func InsertRecord(dst any) error {
	db, err := DB()
	if err != nil {
		return err
	}

	if err := db.Create(dst).Error; err != nil {
		return err
	}

	return nil
}
