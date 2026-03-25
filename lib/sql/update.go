package sql

func Update(tableName string) *Builder {
	b := baseBuilder("UPDATE")
	b.tableName = tableName

	return b
}

func UpdateTable(table TableName) *Builder {
	return Update(renderStaticExpr(table.name))
}

func (b *Builder) Set(vals H) *Builder {
	if vals == nil {
		vals = H{}
	}
	b.vals = vals
	return b
}
