package sql

import "strings"

func Select(fields string) *Builder {
	b := baseBuilder("SELECT")

	if fields != "*" {
		b.fields = strings.Split(fields, ",")
	} else {
		b.fields = []string{"*"}
	}

	return b
}

func (b *Builder) From(tableName string) *Builder {
	b.tableName = tableName
	return b
}
