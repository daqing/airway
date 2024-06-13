package repo

func Count[T TableNameType](conds []KVPair) (n int64, err error) {
	var t T

	db, err := DB()
	if err != nil {
		return 0, err
	}

	db.Table(t.TableName()).Where(buildCondQuery(conds)).Count(&n)

	return n, nil
}
