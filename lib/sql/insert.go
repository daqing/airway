package sql

func Insert(vals H) *Builder {
	b := baseBuilder("INSERT")
	b.vals = vals

	return b
}

func (b *Builder) Into(tableName string) *Builder {
	b.tableName = tableName
	return b
}
