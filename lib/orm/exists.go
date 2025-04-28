package orm

func Exists[T Table](db *DB, cond CondBuilder) (bool, error) {
	n, err := Count[T](db, cond)

	return n > 0, err
}
