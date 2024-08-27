package sql_orm

func Delete[T TableNameType](cond CondBuilder) error {
	var t T

	db, err := DB()
	if err != nil {
		return err
	}

	db.Table(t.TableName()).Where(cond.Cond()).Delete(&t)

	return nil
}

func DeleteByID[T TableNameType](id any) error {
	return Delete[T](Eq("id", id))
}
