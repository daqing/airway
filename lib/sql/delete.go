package sql

func Delete() *Builder {
	return baseBuilder("DELETE")
}

func DeleteFrom(table TableName) *Builder {
	return Delete().From(renderStaticExpr(table.name))
}
