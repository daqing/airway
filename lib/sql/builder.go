package sql

import (
	"fmt"
	"strings"
)

type Builder struct {
	kind      string
	tableName string
	fields    []string
	cond      CondBuilder
	vals      map[string]any
	orderBy   string
	offset    int
	limit     int
}

func baseBuilder(kind string) *Builder {
	return &Builder{
		kind:      kind,
		tableName: "",
		fields:    []string{},
		cond:      EmptyCond{},
		vals:      make(map[string]any),
		offset:    -1,
		limit:     -1,
	}
}

func (b *Builder) Where(cond CondBuilder) *Builder {
	b.cond = cond
	return b
}

func (b *Builder) Limit(limit int) *Builder {
	b.limit = limit
	return b
}

func (b *Builder) Offset(offset int) *Builder {
	b.offset = offset
	return b
}

func (b *Builder) OrderBy(orderBy string) *Builder {
	b.orderBy = orderBy
	return b
}

func (b *Builder) Page(page, perPage int) *Builder {
	if page < 1 {
		page = 1
	}

	b.limit = perPage
	b.offset = (page - 1) * perPage
	return b
}

func (b *Builder) ToSQL() (string, NamedArgs) {
	switch b.kind {
	case "SELECT":
		return b.buildSelect()
	case "INSERT":
		return b.buildInsert()
	case "UPDATE":
		return b.buildUpdate()
	case "DELETE":
		return b.buildDelete()
	default:
		return "", nil
	}
}

func (b *Builder) buildSelect() (string, NamedArgs) {
	sql := fmt.Sprintf("SELECT * FROM %s", b.tableName)
	where, args := b.cond.ToSQL()
	if where != "" {
		sql += " WHERE " + where
	}

	if b.orderBy != "" {
		sql += " ORDER BY " + b.orderBy
	}

	if b.limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", b.limit)
	}

	if b.offset > -1 {
		sql += fmt.Sprintf(" OFFSET %d", b.offset)
	}

	return sql, args
}

func (b *Builder) buildInsert() (string, NamedArgs) {
	if len(b.vals) == 0 {
		panic("no values to insert")
	}

	sql := fmt.Sprintf("INSERT INTO %s", b.tableName)

	columns := make([]string, 0, len(b.vals))
	atColumns := make([]string, 0, len(b.vals))
	for col := range b.vals {
		columns = append(columns, col)
		atColumns = append(atColumns, fmt.Sprintf("@%s", col))
	}

	sql += " (" + strings.Join(columns, ", ") + ")"
	sql += " VALUES (" + strings.Join(atColumns, ", ") + ") RETURNING *"

	return sql, NamedArgs(b.vals)
}

func (b *Builder) buildUpdate() (string, NamedArgs) {
	if len(b.vals) == 0 {
		panic("no values to update")
	}

	sql := fmt.Sprintf("UPDATE %s SET ", b.tableName)
	setClauses := make([]string, 0, len(b.vals))
	args := make(NamedArgs, len(b.vals))
	for col, val := range b.vals {
		setClauses = append(setClauses, fmt.Sprintf("%s = @%s", col, col))
		args[col] = val
	}

	sql += strings.Join(setClauses, ", ")
	where, condArgs := b.cond.ToSQL()
	if where != "" {
		sql += " WHERE " + where
		args = mergeMap(args, condArgs)
	}

	return sql, args
}

func (b *Builder) buildDelete() (string, NamedArgs) {
	sql := fmt.Sprintf("DELETE FROM %s", b.tableName)
	if b.orderBy != "" || b.limit > 0 {
		sql += " WHERE id IN (SELECT id FROM " + b.tableName

		where, args := b.cond.ToSQL()
		if where != "" {
			sql += " WHERE " + where
		}

		if b.orderBy != "" {
			sql += " ORDER BY " + b.orderBy
		}

		if b.limit > 0 {
			sql += fmt.Sprintf(" LIMIT %d", b.limit)
		}

		sql += ")"

		return sql, args
	}

	where, args := b.cond.ToSQL()
	if where != "" {
		sql += " WHERE " + where
	}

	return sql, args
}
