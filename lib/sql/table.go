package sql

type Table interface {
	TableName() string
}

func TableFor(t Table) TableName {
	return Ref(t.TableName())
}

func TableRefOf(t Table) TableName {
	return TableFor(t)
}

func FieldFor(t Table, column string) FieldName {
	return TableFor(t).Field(column)
}

func TableColumn(t Table, column string) FieldName {
	return FieldFor(t, column)
}

func All(t Table) *Builder {
	table := TableFor(t)
	return SelectFields(table.AllFields()).FromTable(table)
}

func FindBy(t Table, vals H) *Builder {
	return FindByCond(t, MatchTable(t, vals))
}

func FindByCond(t Table, cond CondBuilder) *Builder {
	table := TableFor(t)
	return SelectFields(table.AllFields()).FromTable(table).Where(cond)
}

func Create(t Table, vals H) *Builder {
	return Insert(vals).IntoTable(TableFor(t))
}

func UpdateAll(t Table, vals H) *Builder {
	return UpdateTable(TableFor(t)).Set(vals)
}

func DeleteById(t Table, id IdType) *Builder {
	table := TableFor(t)
	return DeleteFrom(table).Where(FieldEq(table.Field("id"), id))
}

func Exists(t Table, vals H) *Builder {
	return SelectColumns("count(*)").FromTable(TableFor(t)).Where(MatchTable(t, vals)).Limit(1)
}
