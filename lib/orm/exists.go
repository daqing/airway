package orm

func Exists[T TableNameType](cond CondBuilder) (bool, error) {
	n, err := Count[T](cond)

	return n > 0, err
}
