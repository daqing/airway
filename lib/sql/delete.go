package sql

func Delete() *Builder {
	return baseBuilder("DELETE")
}
