package sql

func Update(tableName string) *Builder {
	b := baseBuilder("UPDATE")
	b.tableName = tableName

	return b
}

func (b *Builder) Set(vals H) *Builder {
	b.vals = vals
	return b
}
