package repo

import "time"

func UpdateFields[T TableNameType](id uint, fields []KVPair) bool {
	var t T

	db, err := DB()
	if err != nil {
		return false
	}

	now := time.Now().UTC()

	row := buildCondQuery(fields)
	row["updated_at"] = now

	tx := db.Table(t.TableName()).Where("id = ?", id).Updates(row)

	return tx.RowsAffected == 1
}

func UpdateColumn[T TableNameType](cond []KVPair, field string, value any) bool {
	var t T

	db, err := DB()
	if err != nil {
		return false
	}

	tx := db.Table(t.TableName()).Where(buildCondQuery(cond)).Update(field, value)

	return tx.RowsAffected == 1
}
