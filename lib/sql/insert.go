package sql

func Insert(vals H) *Builder {
	b := baseBuilder("INSERT")
	b.vals = vals

	return b
}

func InsertRows(rows ...H) *Builder {
	b := baseBuilder("INSERT")
	b.rows = append([]H{}, rows...)

	return b
}

func (b *Builder) Into(tableName string) *Builder {
	b.tableName = tableName
	return b
}

func (b *Builder) IntoTable(table TableName) *Builder {
	b.tableName = renderStaticExpr(table.name)
	return b
}
