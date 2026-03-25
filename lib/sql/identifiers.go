package sql

import "strings"

type Identifier struct {
	parts []string
}

type TableRef struct {
	name  Identifier
	alias string
}

type ColumnRef struct {
	qualifier Identifier
	name      string
	alias     string
	star      bool
}

type TableName = TableRef

type FieldName = ColumnRef

func TableOf(parts ...string) TableName {
	return TableRef{name: Ident(parts...)}
}

func TableAlias(name string, alias string) TableName {
	return TableOf(name).As(alias)
}

func RefAs(name string, alias string) TableName {
	return TableAlias(name, alias)
}

func Ident(parts ...string) Identifier {
	if len(parts) == 1 {
		parts = strings.Split(parts[0], ".")
	}

	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		normalized = append(normalized, part)
	}

	return Identifier{parts: normalized}
}

func Ref(parts ...string) TableName {
	return TableOf(parts...)
}

func Col(parts ...string) FieldName {
	ident := Ident(parts...)
	if len(ident.parts) == 0 {
		return ColumnRef{}
	}

	if ident.parts[len(ident.parts)-1] == "*" {
		return ColumnRef{
			qualifier: Identifier{parts: append([]string{}, ident.parts[:len(ident.parts)-1]...)},
			star:      true,
		}
	}

	return ColumnRef{
		qualifier: Identifier{parts: append([]string{}, ident.parts[:len(ident.parts)-1]...)},
		name:      ident.parts[len(ident.parts)-1],
	}
}

func Field(parts ...string) FieldName {
	return Col(parts...)
}

func (i Identifier) String() string {
	return i.buildExpr(newBuildState())
}

func (i Identifier) buildExpr(*buildState) string {
	if len(i.parts) == 0 {
		return ""
	}

	quoted := make([]string, 0, len(i.parts))
	for _, part := range i.parts {
		if part == "*" {
			quoted = append(quoted, part)
			continue
		}
		quoted = append(quoted, quoteIdentifier(part))
	}

	return strings.Join(quoted, ".")
}

func (t TableRef) As(alias string) TableRef {
	t.alias = strings.TrimSpace(alias)
	return t
}

func (t TableRef) Column(name string) ColumnRef {
	qualifier := t.name
	if t.alias != "" {
		qualifier = Ident(t.alias)
	}

	return ColumnRef{qualifier: qualifier, name: strings.TrimSpace(name)}
}

func (t TableRef) Col(name string) ColumnRef {
	return t.Column(name)
}

func (t TableRef) Field(name string) ColumnRef {
	return t.Column(name)
}

func (t TableRef) Star() ColumnRef {
	qualifier := t.name
	if t.alias != "" {
		qualifier = Ident(t.alias)
	}

	return ColumnRef{qualifier: qualifier, star: true}
}

func (t TableRef) All() ColumnRef {
	return t.Star()
}

func (t TableRef) AllFields() ColumnRef {
	return t.Star()
}

func (t TableRef) Expr() SQLExpr {
	return ExprFrom(t)
}

func (t TableRef) String() string {
	return t.buildExpr(newBuildState())
}

func (t TableRef) buildExpr(state *buildState) string {
	base := t.name.buildExpr(state)
	if t.alias == "" {
		return base
	}

	return base + " AS " + quoteIdentifier(t.alias)
}

func (c ColumnRef) As(alias string) ColumnRef {
	c.alias = strings.TrimSpace(alias)
	return c
}

func (c ColumnRef) Ref() ColumnRef {
	c.alias = ""
	return c
}

func (c ColumnRef) WithoutAlias() ColumnRef {
	return c.Ref()
}

func (c ColumnRef) Asc() string {
	return c.Ref().String() + " ASC"
}

func (c ColumnRef) Desc() string {
	return c.Ref().String() + " DESC"
}

func (c ColumnRef) Expr() SQLExpr {
	return ExprFrom(c)
}

func (c ColumnRef) String() string {
	return c.buildExpr(newBuildState())
}

func (c ColumnRef) buildExpr(state *buildState) string {
	var builder strings.Builder
	if len(c.qualifier.parts) > 0 {
		builder.WriteString(c.qualifier.buildExpr(state))
		builder.WriteString(".")
	}

	if c.star {
		builder.WriteString("*")
	} else {
		builder.WriteString(quoteIdentifier(c.name))
	}

	if c.alias != "" {
		builder.WriteString(" AS ")
		builder.WriteString(quoteIdentifier(c.alias))
	}

	return builder.String()
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func renderStaticExpr(value compiledExpr) string {
	state := newBuildState()
	result := value.buildExpr(state)
	if len(state.args) > 0 {
		panic("static expression unexpectedly generated bind args")
	}

	return result
}
