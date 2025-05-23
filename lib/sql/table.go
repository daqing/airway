package sql

type Table interface {
	TableName() string
}

func All(t Table) *Builder {
	return Select("*").From(t.TableName())
}

func FindBy(t Table, vals H) *Builder {
	return FindByCond(t, HCond(vals))
}

func FindByCond(t Table, cond CondBuilder) *Builder {
	return Select("*").From(t.TableName()).Where(cond)
}

func Create(t Table, vals H) *Builder {
	return Insert(vals).Into(t.TableName())
}

func UpdateAll(t Table, vals H) *Builder {
	return Update(t.TableName()).Set(vals)
}

func DeleteById(t Table, id IdType) *Builder {
	return Delete().From(t.TableName()).Where(Eq("id", id))
}

func Exists(t Table, vals H) *Builder {
	return Select("count(*)").From(t.TableName()).Where(HCond(vals)).Limit(1)
}
