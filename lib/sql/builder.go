package sql

import (
	"fmt"
	"strings"
)

type Builder struct {
	kind        string
	tableName   string
	fromExpr    *SQLExpr
	fields      []string
	insertCols  []string
	insertQuery *Builder
	cond        CondBuilder
	vals        H
	rows        []H
	orderBys    []string
	groupBys    []string
	windows     []string
	having      CondBuilder
	joins       []joinClause
	setOps      []setOperation
	withClauses []cteClause
	returning   []string
	updateFrom  []string
	usingTables []string
	distinct    bool
	distinctOn  []string
	lockClause  string
	conflict    *conflictClause
	deleteKey   string
	offset      int
	limit       int
}

type cteClause struct {
	name      string
	query     *Builder
	recursive bool
}

type joinClause struct {
	kind      string
	tableName string
	expr      *SQLExpr
	lateral   bool
	on        CondBuilder
}

type setOperation struct {
	kind  string
	query *Builder
}

type conflictClause struct {
	target     []string
	constraint string
	doNothing  bool
	set        H
}

func baseBuilder(kind string) *Builder {
	return &Builder{
		kind:      kind,
		tableName: "",
		fields:    []string{},
		cond:      EmptyCond{},
		vals:      H{},
		deleteKey: "id",
		offset:    -1,
		limit:     -1,
	}
}

func (b *Builder) Where(cond CondBuilder) *Builder {
	if cond == nil {
		cond = EmptyCond{}
	}

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
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" {
		return b
	}

	b.orderBys = append(b.orderBys, orderBy)
	return b
}

func (b *Builder) OrderBys(orderBys ...string) *Builder {
	for _, orderBy := range orderBys {
		b.OrderBy(orderBy)
	}

	return b
}

func (b *Builder) GroupBy(fields ...string) *Builder {
	b.groupBys = append(b.groupBys, normalizeFields(fields)...)
	return b
}

func (b *Builder) GroupByFields(columns ...FieldName) *Builder {
	for _, column := range columns {
		b.groupBys = append(b.groupBys, renderStaticExpr(column.Ref()))
	}

	return b
}

func (b *Builder) GroupByRefs(columns ...FieldName) *Builder {
	return b.GroupByFields(columns...)
}

func (b *Builder) Window(definitions ...string) *Builder {
	b.windows = append(b.windows, normalizeFields(definitions)...)
	return b
}

func (b *Builder) Having(cond CondBuilder) *Builder {
	b.having = cond
	return b
}

func (b *Builder) Distinct() *Builder {
	b.distinct = true
	return b
}

func (b *Builder) DistinctOn(fields ...string) *Builder {
	b.distinctOn = append([]string{}, normalizeFields(fields)...)
	return b
}

func (b *Builder) Returning(fields ...string) *Builder {
	b.returning = append([]string{}, normalizeFields(fields)...)
	return b
}

func (b *Builder) ReturningFields(columns ...FieldName) *Builder {
	b.returning = []string{}
	for _, column := range columns {
		b.returning = append(b.returning, renderStaticExpr(column.Ref()))
	}

	return b
}

func (b *Builder) ReturningRefs(columns ...FieldName) *Builder {
	return b.ReturningFields(columns...)
}

func (b *Builder) ReturningAll() *Builder {
	b.returning = []string{"*"}
	return b
}

func (b *Builder) For(clause string) *Builder {
	clause = strings.TrimSpace(clause)
	if clause == "" {
		return b
	}

	b.lockClause = clause
	return b
}

func (b *Builder) ForUpdate() *Builder {
	return b.For("FOR UPDATE")
}

func (b *Builder) ForShare() *Builder {
	return b.For("FOR SHARE")
}

func (b *Builder) With(name string, query *Builder) *Builder {
	b.withClauses = append(b.withClauses, cteClause{name: strings.TrimSpace(name), query: query})
	return b
}

func (b *Builder) WithRecursive(name string, query *Builder) *Builder {
	b.withClauses = append(b.withClauses, cteClause{name: strings.TrimSpace(name), query: query, recursive: true})
	return b
}

func (b *Builder) Join(tableName string, on CondBuilder) *Builder {
	b.joins = append(b.joins, joinClause{kind: "JOIN", tableName: strings.TrimSpace(tableName), on: on})
	return b
}

func (b *Builder) JoinTable(table TableName, on CondBuilder) *Builder {
	expr := table.Expr()
	b.joins = append(b.joins, joinClause{kind: "JOIN", expr: &expr, on: on})
	return b
}

func (b *Builder) LeftJoin(tableName string, on CondBuilder) *Builder {
	b.joins = append(b.joins, joinClause{kind: "LEFT JOIN", tableName: strings.TrimSpace(tableName), on: on})
	return b
}

func (b *Builder) LeftJoinTable(table TableName, on CondBuilder) *Builder {
	expr := table.Expr()
	b.joins = append(b.joins, joinClause{kind: "LEFT JOIN", expr: &expr, on: on})
	return b
}

func (b *Builder) RightJoin(tableName string, on CondBuilder) *Builder {
	b.joins = append(b.joins, joinClause{kind: "RIGHT JOIN", tableName: strings.TrimSpace(tableName), on: on})
	return b
}

func (b *Builder) RightJoinTable(table TableName, on CondBuilder) *Builder {
	expr := table.Expr()
	b.joins = append(b.joins, joinClause{kind: "RIGHT JOIN", expr: &expr, on: on})
	return b
}

func (b *Builder) FullJoin(tableName string, on CondBuilder) *Builder {
	b.joins = append(b.joins, joinClause{kind: "FULL JOIN", tableName: strings.TrimSpace(tableName), on: on})
	return b
}

func (b *Builder) FullJoinTable(table TableName, on CondBuilder) *Builder {
	expr := table.Expr()
	b.joins = append(b.joins, joinClause{kind: "FULL JOIN", expr: &expr, on: on})
	return b
}

func (b *Builder) CrossJoin(tableName string) *Builder {
	b.joins = append(b.joins, joinClause{kind: "CROSS JOIN", tableName: strings.TrimSpace(tableName)})
	return b
}

func (b *Builder) CrossJoinTable(table TableName) *Builder {
	expr := table.Expr()
	b.joins = append(b.joins, joinClause{kind: "CROSS JOIN", expr: &expr})
	return b
}

func (b *Builder) JoinExpr(expr SQLExpr, on CondBuilder) *Builder {
	b.joins = append(b.joins, joinClause{kind: "JOIN", expr: &expr, on: on})
	return b
}

func (b *Builder) JoinLateral(query *Builder, alias string, on CondBuilder) *Builder {
	return b.joinLateral("JOIN", query, alias, on)
}

func (b *Builder) LeftJoinLateral(query *Builder, alias string, on CondBuilder) *Builder {
	return b.joinLateral("LEFT JOIN", query, alias, on)
}

func (b *Builder) joinLateral(kind string, query *Builder, alias string, on CondBuilder) *Builder {
	expr := SubQuery(query)
	if alias = strings.TrimSpace(alias); alias != "" {
		expr.SQL += " AS " + alias
	}

	b.joins = append(b.joins, joinClause{kind: kind, expr: &expr, lateral: true, on: on})
	return b
}

func (b *Builder) UpdateFrom(tables ...string) *Builder {
	b.updateFrom = append(b.updateFrom, normalizeFields(tables)...)
	return b
}

func (b *Builder) Using(tables ...string) *Builder {
	b.usingTables = append(b.usingTables, normalizeFields(tables)...)
	return b
}

func (b *Builder) Values(rows ...H) *Builder {
	b.rows = append([]H{}, rows...)
	return b
}

func (b *Builder) InsertColumns(columns ...string) *Builder {
	b.insertCols = append([]string{}, normalizeFields(columns)...)
	return b
}

func (b *Builder) FromSelect(query *Builder, columns ...string) *Builder {
	b.insertQuery = query
	if len(columns) > 0 {
		b.insertCols = append([]string{}, normalizeFields(columns)...)
	}
	return b
}

func (b *Builder) DeleteKey(column string) *Builder {
	column = strings.TrimSpace(column)
	if column != "" {
		b.deleteKey = column
	}
	return b
}

func (b *Builder) DeleteKeyField(column FieldName) *Builder {
	b.deleteKey = renderStaticExpr(column.Ref())
	return b
}

func (b *Builder) DeleteKeyRef(column FieldName) *Builder {
	return b.DeleteKeyField(column)
}

func (b *Builder) OnConflictDoNothing(columns ...string) *Builder {
	b.conflict = &conflictClause{
		target:    normalizeFields(columns),
		doNothing: true,
	}
	return b
}

func (b *Builder) OnConflictOnConstraintDoNothing(constraint string) *Builder {
	b.conflict = &conflictClause{
		constraint: strings.TrimSpace(constraint),
		doNothing:  true,
	}
	return b
}

func (b *Builder) OnConflictDoUpdate(columns []string, vals H) *Builder {
	b.conflict = &conflictClause{
		target: normalizeFields(columns),
		set:    vals,
	}
	return b
}

func (b *Builder) OnConflictOnConstraintDoUpdate(constraint string, vals H) *Builder {
	b.conflict = &conflictClause{
		constraint: strings.TrimSpace(constraint),
		set:        vals,
	}
	return b
}

func (b *Builder) Union(query *Builder) *Builder {
	b.setOps = append(b.setOps, setOperation{kind: "UNION", query: query})
	return b
}

func (b *Builder) UnionAll(query *Builder) *Builder {
	b.setOps = append(b.setOps, setOperation{kind: "UNION ALL", query: query})
	return b
}

func (b *Builder) Intersect(query *Builder) *Builder {
	b.setOps = append(b.setOps, setOperation{kind: "INTERSECT", query: query})
	return b
}

func (b *Builder) IntersectAll(query *Builder) *Builder {
	b.setOps = append(b.setOps, setOperation{kind: "INTERSECT ALL", query: query})
	return b
}

func (b *Builder) Except(query *Builder) *Builder {
	b.setOps = append(b.setOps, setOperation{kind: "EXCEPT", query: query})
	return b
}

func (b *Builder) ExceptAll(query *Builder) *Builder {
	b.setOps = append(b.setOps, setOperation{kind: "EXCEPT ALL", query: query})
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
	state := newBuildState()

	var sql string
	switch b.kind {
	case "SELECT":
		sql = b.buildSelect(state)
	case "INSERT":
		sql = b.buildInsert(state)
	case "UPDATE":
		sql = b.buildUpdate(state)
	case "DELETE":
		sql = b.buildDelete(state)
	default:
		return "", nil
	}

	return sql, state.args
}

func (b *Builder) Kind() string {
	if b == nil {
		return ""
	}

	return b.kind
}

func (b *Builder) TableName() string {
	if b == nil {
		return ""
	}

	return b.tableName
}

func (b *Builder) InsertValues() H {
	if b == nil || len(b.vals) == 0 {
		return nil
	}

	vals := H{}
	for key, value := range b.vals {
		vals[key] = value
	}

	return vals
}

func (b *Builder) InsertRows() []H {
	if b == nil || len(b.rows) == 0 {
		return nil
	}

	rows := make([]H, 0, len(b.rows))
	for _, row := range b.rows {
		copied := H{}
		for key, value := range row {
			copied[key] = value
		}
		rows = append(rows, copied)
	}

	return rows
}

func (b *Builder) ConflictTarget() []string {
	if b == nil || b.conflict == nil || len(b.conflict.target) == 0 {
		return nil
	}

	target := make([]string, 0, len(b.conflict.target))
	target = append(target, b.conflict.target...)
	return target
}

func (b *Builder) buildSelect(state *buildState) string {
	baseSQL := b.buildSelectCore(state)
	if len(b.setOps) == 0 {
		var sql strings.Builder
		sql.WriteString(baseSQL)
		b.writeOrderLimitOffset(&sql)
		if b.lockClause != "" {
			sql.WriteString(" ")
			sql.WriteString(b.lockClause)
		}

		return sql.String()
	}
	var sql strings.Builder
	sql.WriteString(baseSQL)
	for _, setOp := range b.setOps {
		if setOp.query == nil {
			continue
		}
		querySQL, queryArgs := setOp.query.ToSQL()
		sql.WriteString(" ")
		sql.WriteString(setOp.kind)
		sql.WriteString(" (")
		sql.WriteString(state.mergeExpr(querySQL, queryArgs))
		sql.WriteString(")")
	}

	b.writeOrderLimitOffset(&sql)
	if b.lockClause != "" {
		sql.WriteString(" ")
		sql.WriteString(b.lockClause)
	}

	return sql.String()
}

func (b *Builder) buildSelectCore(state *buildState) string {
	fields := append([]string{}, b.fields...)
	if len(fields) == 0 {
		fields = []string{"*"}
	}

	var sql strings.Builder
	b.writeWithClause(&sql, state)
	sql.WriteString("SELECT ")

	switch {
	case len(b.distinctOn) > 0:
		sql.WriteString("DISTINCT ON (")
		sql.WriteString(strings.Join(b.distinctOn, ", "))
		sql.WriteString(") ")
	case b.distinct:
		sql.WriteString("DISTINCT ")
	}

	sql.WriteString(strings.Join(fields, ", "))

	fromSQL := b.buildFrom(state)
	if fromSQL != "" {
		sql.WriteString(" FROM ")
		sql.WriteString(fromSQL)
	}

	for _, join := range b.joins {
		joinSQL := join.build(state)
		if joinSQL == "" {
			continue
		}
		sql.WriteString(" ")
		sql.WriteString(joinSQL)
	}

	where := compileCond(b.cond, state)
	if where != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(where)
	}

	if len(b.groupBys) > 0 {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(b.groupBys, ", "))
	}

	having := compileCond(b.having, state)
	if having != "" {
		sql.WriteString(" HAVING ")
		sql.WriteString(having)
	}

	if len(b.windows) > 0 {
		sql.WriteString(" WINDOW ")
		sql.WriteString(strings.Join(b.windows, ", "))
	}

	return sql.String()
}

func (b *Builder) buildInsert(state *buildState) string {
	if b.insertQuery != nil {
		return b.buildInsertSelect(state)
	}

	rows := b.rows
	if len(rows) == 0 {
		rows = []H{b.vals}
	}

	if len(rows) == 0 || len(rows[0]) == 0 {
		panic("no values to insert")
	}

	columns := sortedMapKeys(rows[0])
	for idx, row := range rows {
		if len(row) != len(columns) {
			panic(fmt.Sprintf("insert row %d does not match column count", idx))
		}
		for _, column := range columns {
			if _, ok := row[column]; !ok {
				panic(fmt.Sprintf("insert row %d is missing column %s", idx, column))
			}
		}
	}

	var sql strings.Builder
	b.writeWithClause(&sql, state)
	sql.WriteString(fmt.Sprintf("INSERT INTO %s (%s)", b.tableName, strings.Join(columns, ", ")))
	sql.WriteString(" VALUES ")

	rowClauses := make([]string, 0, len(rows))
	for _, row := range rows {
		values := make([]string, 0, len(columns))
		for _, column := range columns {
			values = append(values, renderValue(state, column, row[column]))
		}
		rowClauses = append(rowClauses, "("+strings.Join(values, ", ")+")")
	}

	sql.WriteString(strings.Join(rowClauses, ", "))
	b.writeConflictClause(&sql, state)
	b.writeReturningClause(&sql, true)

	return sql.String()
}

func (b *Builder) buildInsertSelect(state *buildState) string {
	if b.insertQuery == nil {
		panic("no select query to insert from")
	}

	if len(b.insertCols) == 0 {
		panic("insert-select requires explicit insert columns")
	}

	querySQL, queryArgs := b.insertQuery.ToSQL()

	var sql strings.Builder
	b.writeWithClause(&sql, state)
	sql.WriteString(fmt.Sprintf("INSERT INTO %s (%s) ", b.tableName, strings.Join(b.insertCols, ", ")))
	sql.WriteString(state.mergeExpr(querySQL, queryArgs))
	b.writeConflictClause(&sql, state)
	b.writeReturningClause(&sql, true)

	return sql.String()
}

func (b *Builder) buildUpdate(state *buildState) string {
	if len(b.vals) == 0 {
		panic("no values to update")
	}

	var sql strings.Builder
	b.writeWithClause(&sql, state)
	sql.WriteString(fmt.Sprintf("UPDATE %s SET ", b.tableName))

	setClauses := make([]string, 0, len(b.vals))
	for _, column := range sortedMapKeys(b.vals) {
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", column, renderValue(state, column, b.vals[column])))
	}
	sql.WriteString(strings.Join(setClauses, ", "))

	if len(b.updateFrom) > 0 {
		sql.WriteString(" FROM ")
		sql.WriteString(strings.Join(b.updateFrom, ", "))
	}

	where := compileCond(b.cond, state)
	if where != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(where)
	}

	b.writeReturningClause(&sql, false)

	return sql.String()
}

func (b *Builder) buildDelete(state *buildState) string {
	var sql strings.Builder
	b.writeWithClause(&sql, state)
	sql.WriteString(fmt.Sprintf("DELETE FROM %s", b.tableName))

	if len(b.usingTables) > 0 {
		sql.WriteString(" USING ")
		sql.WriteString(strings.Join(b.usingTables, ", "))
	}

	if len(b.orderBys) > 0 || b.limit > -1 || b.offset > -1 {
		sql.WriteString(fmt.Sprintf(" WHERE %s IN (SELECT %s FROM %s", b.deleteKey, b.deleteKey, b.tableName))

		where := compileCond(b.cond, state)
		if where != "" {
			sql.WriteString(" WHERE ")
			sql.WriteString(where)
		}

		if len(b.orderBys) > 0 {
			sql.WriteString(" ORDER BY ")
			sql.WriteString(strings.Join(b.orderBys, ", "))
		}

		if b.limit > -1 {
			sql.WriteString(fmt.Sprintf(" LIMIT %d", b.limit))
		}

		if b.offset > -1 {
			sql.WriteString(fmt.Sprintf(" OFFSET %d", b.offset))
		}

		sql.WriteString(")")
		b.writeReturningClause(&sql, false)
		return sql.String()
	}

	where := compileCond(b.cond, state)
	if where != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(where)
	}

	b.writeReturningClause(&sql, false)

	return sql.String()
}

func (b *Builder) writeWithClause(sql *strings.Builder, state *buildState) {
	if len(b.withClauses) == 0 {
		return
	}

	parts := make([]string, 0, len(b.withClauses))
	recursive := false
	for _, cte := range b.withClauses {
		if cte.name == "" || cte.query == nil {
			continue
		}

		recursive = recursive || cte.recursive
		querySQL, queryArgs := cte.query.ToSQL()
		parts = append(parts, fmt.Sprintf("%s AS (%s)", cte.name, state.mergeExpr(querySQL, queryArgs)))
	}

	if len(parts) == 0 {
		return
	}

	sql.WriteString("WITH ")
	if recursive {
		sql.WriteString("RECURSIVE ")
	}
	sql.WriteString(strings.Join(parts, ", "))
	sql.WriteString(" ")
}

func (b *Builder) buildFrom(state *buildState) string {
	if b.fromExpr != nil {
		return b.fromExpr.buildExpr(state)
	}

	return b.tableName
}

func (b *Builder) writeOrderLimitOffset(sql *strings.Builder) {
	if len(b.orderBys) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(b.orderBys, ", "))
	}

	if b.limit > -1 {
		sql.WriteString(fmt.Sprintf(" LIMIT %d", b.limit))
	}

	if b.offset > -1 {
		sql.WriteString(fmt.Sprintf(" OFFSET %d", b.offset))
	}
}

func (b *Builder) writeReturningClause(sql *strings.Builder, insertDefault bool) {
	fields := b.returning
	if insertDefault && len(fields) == 0 {
		fields = []string{"*"}
	}

	if len(fields) == 0 {
		return
	}

	sql.WriteString(" RETURNING ")
	sql.WriteString(strings.Join(fields, ", "))
}

func (b *Builder) writeConflictClause(sql *strings.Builder, state *buildState) {
	if b.conflict == nil {
		return
	}

	sql.WriteString(" ON CONFLICT")
	switch {
	case b.conflict.constraint != "":
		sql.WriteString(" ON CONSTRAINT ")
		sql.WriteString(b.conflict.constraint)
	case len(b.conflict.target) > 0:
		sql.WriteString(" (")
		sql.WriteString(strings.Join(b.conflict.target, ", "))
		sql.WriteString(")")
	}

	if b.conflict.doNothing {
		sql.WriteString(" DO NOTHING")
		return
	}

	if len(b.conflict.set) == 0 {
		panic("on conflict do update requires values")
	}

	clauses := make([]string, 0, len(b.conflict.set))
	for _, column := range sortedMapKeys(b.conflict.set) {
		clauses = append(clauses, fmt.Sprintf("%s = %s", column, renderValue(state, column, b.conflict.set[column])))
	}

	sql.WriteString(" DO UPDATE SET ")
	sql.WriteString(strings.Join(clauses, ", "))
}

func (j joinClause) build(state *buildState) string {
	source := j.tableName
	if j.expr != nil {
		source = j.expr.buildExpr(state)
	}

	if source == "" {
		return ""
	}

	if j.lateral {
		source = "LATERAL " + source
	}

	if j.kind == "CROSS JOIN" {
		return j.kind + " " + source
	}

	joinSQL := j.kind + " " + source
	on := compileCond(j.on, state)
	if on != "" {
		joinSQL += " ON " + on
	}

	return joinSQL
}
