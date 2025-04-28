package orm

func Page[T Table](db *DB, fields []string, order string, page, limit int) (all []*T, total int64, err error) {
	if page == 0 {
		page = 1
	}

	cond := EmptyCond{}

	all, err = FindLimit[T](
		db,
		fields,
		cond,
		order,
		(page-1)*limit,
		limit,
	)

	if err != nil {
		return nil, 0, err
	}

	total, err = Count[T](db, cond)

	return
}
