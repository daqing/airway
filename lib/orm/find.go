package orm

import "gorm.io/gorm"

func FindOne[T Table](db *gorm.DB, fields []string, cond CondBuilder) (*T, error) {
	rows, err := Find[T](db, fields, cond)
	if err != nil {
		return nil, err
	}

	if len(rows) > 1 {
		return nil, ErrorCountNotMatch
	}

	if len(rows) == 0 {
		return nil, ErrorNotFound
	}

	return rows[0], nil
}

func FindAll[T Table](db *gorm.DB, fields []string) ([]*T, error) {
	return Find[T](db, fields, EmptyCond{})
}

func Find[T Table](db *gorm.DB, fields []string, cond CondBuilder) ([]*T, error) {
	return FindLimit[T](db, fields, cond, "", 0, 0)
}

// limit = 0 means no limit
func FindLimit[T Table](db *gorm.DB, fields []string, cond CondBuilder, orderBy string, offset int, limit int) ([]*T, error) {
	var t T

	tx := db.Table(t.TableName()).Select(fields).Where(cond.Cond())

	if orderBy != EMPTY_STRING {
		tx = tx.Order(orderBy)
	}

	if limit > 0 {
		tx = tx.Limit(limit)
	}

	if offset > 0 {
		tx = tx.Offset(offset)
	}

	var records []*T

	tx.Find(&records)

	return records, nil

}
