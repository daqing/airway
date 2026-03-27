package sql

import "strings"

func Select(fields string) *Builder {
	parts := []string{"*"}
	if strings.TrimSpace(fields) != "" && strings.TrimSpace(fields) != "*" {
		parts = strings.Split(fields, ",")
	}

	return SelectColumns(parts...)
}

func SelectColumns(fields ...string) *Builder {
	b := baseBuilder("SELECT")
	b.fields = normalizeFields(fields)
	if len(b.fields) == 0 {
		b.fields = []string{"*"}
	}

	return b
}

func SelectFields(columns ...FieldName) *Builder {
	fields := make([]string, 0, len(columns))
	for _, column := range columns {
		fields = append(fields, renderStaticExpr(column))
	}

	return SelectColumns(fields...)
}

func SelectRefs(columns ...FieldName) *Builder {
	return SelectFields(columns...)
}

func (b *Builder) Columns(fields ...string) *Builder {
	b.fields = append(b.fields, normalizeFields(fields)...)
	return b
}

func (b *Builder) Fields(columns ...FieldName) *Builder {
	for _, column := range columns {
		b.fields = append(b.fields, renderStaticExpr(column))
	}

	return b
}

func (b *Builder) ColumnsRefs(columns ...FieldName) *Builder {
	return b.Fields(columns...)
}

func (b *Builder) From(tableName string) *Builder {
	b.fromExpr = nil
	b.tableName = tableName
	return b
}

func (b *Builder) FromTable(table TableName) *Builder {
	return b.FromExpr(table.Expr())
}

func (b *Builder) FromExpr(expr SQLExpr) *Builder {
	b.tableName = ""
	b.fromExpr = &expr
	return b
}

func (b *Builder) FromSubQuery(query *Builder, alias string) *Builder {
	expr := SubQuery(query)
	if strings.TrimSpace(alias) != "" {
		expr.SQL += " AS " + strings.TrimSpace(alias)
	}

	return b.FromExpr(expr)
}
