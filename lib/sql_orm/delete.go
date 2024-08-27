package sql_orm

func Delete[T TableNameType](conds []KVPair) error {
	var t T

	db, err := DB()
	if err != nil {
		return err
	}

	db.Table(t.TableName()).Where(buildCondQuery(conds)).Delete(&t)

	return nil
}

func DeleteByID[T TableNameType](id any) error {
	return Delete[T]([]KVPair{KV("id", id)})
}
