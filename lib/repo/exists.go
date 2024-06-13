package repo

func Exists[T TableNameType](conds []KVPair) (bool, error) {
	n, err := Count[T](conds)

	return n > 0, err
}
