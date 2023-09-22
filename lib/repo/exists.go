package repo

func Exists[T TableNameType](conds []KeyValueField) (bool, error) {
	n, err := Count[T](conds)

	return n > 0, err
}
