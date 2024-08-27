package orm

func Count[T TableNameType](cond CondBuilder) (n int64, err error) {
	var t T

	db, err := DB()
	if err != nil {
		return 0, err
	}

	db.Table(t.TableName()).Where(cond.Cond()).Count(&n)

	return n, nil
}
