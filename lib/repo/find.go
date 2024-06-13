package repo

func FindRow[T TableNameType](fields []string, conds []KVPair) (*T, error) {
	rows, err := Find[T](fields, conds)
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

func FindAll[T TableNameType](fields []string) ([]*T, error) {
	return Find[T](fields, []KVPair{})
}

func Find[T TableNameType](fields []string, conds []KVPair) ([]*T, error) {
	return FindLimit[T](fields, conds, "", 0, 0)
}

// limit = 0 means no limit
func FindLimit[T TableNameType](fields []string, conds []KVPair, orderBy string, offset int, limit int) ([]*T, error) {
	var t T

	db, err := DB()
	if err != nil {
		return nil, err
	}

	tx := db.Table(t.TableName()).Select(fields).Where(buildCondQuery(conds))

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
