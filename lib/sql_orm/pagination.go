package sql_orm

func Page[T TableNameType](fields []string, order string, page, limit int) (all []*T, total int64, err error) {
	if page == 0 {
		page = 1
	}

	cond := []KVPair{}

	all, err = FindLimit[T](
		fields,
		cond,
		order,
		(page-1)*limit,
		limit,
	)

	if err != nil {
		return nil, 0, err
	}

	total, err = Count[T](cond)

	return
}