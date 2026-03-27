// Package mysql provides a MYSQL-specific SQL builder.
//
// Not available: RETURNING, DISTINCT ON, FULL JOIN, LATERAL JOIN, ILIKE,
// FOR SHARE, UPDATE...FROM, DELETE...USING, ON CONFLICT ON CONSTRAINT.
// ON CONFLICT DO NOTHING -> INSERT IGNORE (transformed by lib/repo).
// ON CONFLICT DO UPDATE -> ON DUPLICATE KEY UPDATE (transformed by lib/repo).
package mysql

import (
	sql "github.com/daqing/airway/lib/sql"
)

// Builder is the mysql-specific SQL builder.
type Builder struct {
	inner *sql.Builder
}

func wrap(b *sql.Builder) *Builder { return &Builder{inner: b} }

func (b *Builder) ToSQL() (string, sql.NamedArgs) { return b.inner.ToSQL() }
func (b *Builder) Kind() string                    { return b.inner.Kind() }
func (b *Builder) TableName() string               { return b.inner.TableName() }
func (b *Builder) InsertValues() sql.H             { return b.inner.InsertValues() }
func (b *Builder) InsertRows() []sql.H             { return b.inner.InsertRows() }
func (b *Builder) ConflictTarget() []string        { return b.inner.ConflictTarget() }

// --- SELECT ---

func Select(fields string) *Builder                  { return wrap(sql.Select(fields)) }
func SelectColumns(fields ...string) *Builder        { return wrap(sql.SelectColumns(fields...)) }
func SelectFields(columns ...sql.FieldName) *Builder { return wrap(sql.SelectFields(columns...)) }
func SelectRefs(columns ...sql.FieldName) *Builder   { return SelectFields(columns...) }

func (b *Builder) Columns(fields ...string) *Builder {
	b.inner.Columns(fields...)
	return b
}
func (b *Builder) Fields(columns ...sql.FieldName) *Builder {
	b.inner.Fields(columns...)
	return b
}
func (b *Builder) From(tableName string) *Builder {
	b.inner.From(tableName)
	return b
}
func (b *Builder) FromTable(table sql.TableName) *Builder {
	b.inner.FromTable(table)
	return b
}
func (b *Builder) FromExpr(expr sql.SQLExpr) *Builder {
	b.inner.FromExpr(expr)
	return b
}
func (b *Builder) FromSubQuery(query *Builder, alias string) *Builder {
	b.inner.FromSubQuery(query.inner, alias)
	return b
}

// --- WHERE / ORDER / GROUP / LIMIT ---

func (b *Builder) Where(cond sql.CondBuilder) *Builder {
	b.inner.Where(cond)
	return b
}
func (b *Builder) Limit(limit int) *Builder {
	b.inner.Limit(limit)
	return b
}
func (b *Builder) Offset(offset int) *Builder {
	b.inner.Offset(offset)
	return b
}
func (b *Builder) Page(page, perPage int) *Builder {
	b.inner.Page(page, perPage)
	return b
}
func (b *Builder) OrderBy(orderBy string) *Builder {
	b.inner.OrderBy(orderBy)
	return b
}
func (b *Builder) OrderBys(orderBys ...string) *Builder {
	b.inner.OrderBys(orderBys...)
	return b
}
func (b *Builder) GroupBy(fields ...string) *Builder {
	b.inner.GroupBy(fields...)
	return b
}
func (b *Builder) GroupByFields(columns ...sql.FieldName) *Builder {
	b.inner.GroupByFields(columns...)
	return b
}
func (b *Builder) GroupByRefs(columns ...sql.FieldName) *Builder { return b.GroupByFields(columns...) }
func (b *Builder) Having(cond sql.CondBuilder) *Builder {
	b.inner.Having(cond)
	return b
}
func (b *Builder) Distinct() *Builder {
	b.inner.Distinct()
	return b
}
func (b *Builder) Window(definitions ...string) *Builder {
	b.inner.Window(definitions...)
	return b
}

// --- JOIN ---

func (b *Builder) Join(tableName string, on sql.CondBuilder) *Builder {
	b.inner.Join(tableName, on)
	return b
}
func (b *Builder) JoinTable(table sql.TableName, on sql.CondBuilder) *Builder {
	b.inner.JoinTable(table, on)
	return b
}
func (b *Builder) LeftJoin(tableName string, on sql.CondBuilder) *Builder {
	b.inner.LeftJoin(tableName, on)
	return b
}
func (b *Builder) LeftJoinTable(table sql.TableName, on sql.CondBuilder) *Builder {
	b.inner.LeftJoinTable(table, on)
	return b
}
func (b *Builder) RightJoin(tableName string, on sql.CondBuilder) *Builder {
	b.inner.RightJoin(tableName, on)
	return b
}
func (b *Builder) RightJoinTable(table sql.TableName, on sql.CondBuilder) *Builder {
	b.inner.RightJoinTable(table, on)
	return b
}
func (b *Builder) CrossJoin(tableName string) *Builder {
	b.inner.CrossJoin(tableName)
	return b
}
func (b *Builder) CrossJoinTable(table sql.TableName) *Builder {
	b.inner.CrossJoinTable(table)
	return b
}
func (b *Builder) JoinExpr(expr sql.SQLExpr, on sql.CondBuilder) *Builder {
	b.inner.JoinExpr(expr, on)
	return b
}

// --- CTE ---

func (b *Builder) With(name string, query *Builder) *Builder {
	b.inner.With(name, query.inner)
	return b
}
func (b *Builder) WithRecursive(name string, query *Builder) *Builder {
	b.inner.WithRecursive(name, query.inner)
	return b
}

// --- Set operations ---

func (b *Builder) Union(query *Builder) *Builder {
	b.inner.Union(query.inner)
	return b
}
func (b *Builder) UnionAll(query *Builder) *Builder {
	b.inner.UnionAll(query.inner)
	return b
}
func (b *Builder) Intersect(query *Builder) *Builder {
	b.inner.Intersect(query.inner)
	return b
}
func (b *Builder) Except(query *Builder) *Builder {
	b.inner.Except(query.inner)
	return b
}

// --- INSERT ---

func Insert(vals sql.H) *Builder        { return wrap(sql.Insert(vals)) }
func InsertRows(rows ...sql.H) *Builder { return wrap(sql.InsertRows(rows...)) }

func (b *Builder) Into(tableName string) *Builder {
	b.inner.Into(tableName)
	return b
}
func (b *Builder) IntoTable(table sql.TableName) *Builder {
	b.inner.IntoTable(table)
	return b
}
func (b *Builder) Values(rows ...sql.H) *Builder {
	b.inner.Values(rows...)
	return b
}
func (b *Builder) InsertColumns(columns ...string) *Builder {
	b.inner.InsertColumns(columns...)
	return b
}
func (b *Builder) FromSelect(query *Builder, columns ...string) *Builder {
	b.inner.FromSelect(query.inner, columns...)
	return b
}
func (b *Builder) OnConflictDoNothing(columns ...string) *Builder {
	b.inner.OnConflictDoNothing(columns...)
	return b
}
func (b *Builder) OnConflictDoUpdate(columns []string, vals sql.H) *Builder {
	b.inner.OnConflictDoUpdate(columns, vals)
	return b
}

// --- UPDATE ---

func Update(tableName string) *Builder         { return wrap(sql.Update(tableName)) }
func UpdateTable(table sql.TableName) *Builder { return wrap(sql.UpdateTable(table)) }

func (b *Builder) Set(vals sql.H) *Builder {
	b.inner.Set(vals)
	return b
}

// --- DELETE ---

func Delete() *Builder                        { return wrap(sql.Delete()) }
func DeleteFrom(table sql.TableName) *Builder { return wrap(sql.DeleteFrom(table)) }

func (b *Builder) DeleteKey(column string) *Builder {
	b.inner.DeleteKey(column)
	return b
}
func (b *Builder) DeleteKeyField(column sql.FieldName) *Builder {
	b.inner.DeleteKeyField(column)
	return b
}

// --- Table / model helpers ---

func All(t sql.Table) *Builder                              { return wrap(sql.All(t)) }
func FindBy(t sql.Table, vals sql.H) *Builder               { return wrap(sql.FindBy(t, vals)) }
func FindByCond(t sql.Table, cond sql.CondBuilder) *Builder { return wrap(sql.FindByCond(t, cond)) }
func Create(t sql.Table, vals sql.H) *Builder               { return wrap(sql.Create(t, vals)) }
func UpdateAll(t sql.Table, vals sql.H) *Builder            { return wrap(sql.UpdateAll(t, vals)) }
func DeleteById(t sql.Table, id sql.IdType) *Builder        { return wrap(sql.DeleteById(t, id)) }
func ExistsWhere(t sql.Table, vals sql.H) *Builder          { return wrap(sql.Exists(t, vals)) }

// --- Identifier helpers ---

var (
	TableOf     = sql.TableOf
	TableAlias  = sql.TableAlias
	Ref         = sql.Ref
	RefAs       = sql.RefAs
	Field       = sql.Field
	Col         = sql.Col
	Ident       = sql.Ident
	TableFor    = sql.TableFor
	TableRefOf  = sql.TableRefOf
	FieldFor    = sql.FieldFor
	TableColumn = sql.TableColumn
)

// --- Condition constructors ---

var (
	Eq           = sql.Eq
	NotEq        = sql.NotEq
	Gt           = sql.Gt
	Gte          = sql.Gte
	Lt           = sql.Lt
	Lte          = sql.Lte
	Like         = sql.Like
	NotLike      = sql.NotLike
	AllOf        = sql.AllOf
	AnyOf        = sql.AnyOf
	Not          = sql.Not
	IsNull       = sql.IsNull
	IsNotNull    = sql.IsNotNull
	Between      = sql.Between
	NotBetween   = sql.NotBetween
	HCond        = sql.HCond
	Compare      = sql.Compare
	FieldEq      = sql.FieldEq
	EqRef        = sql.EqRef
	FieldNotEq   = sql.FieldNotEq
	FieldGt      = sql.FieldGt
	FieldGte     = sql.FieldGte
	FieldLt      = sql.FieldLt
	FieldLte     = sql.FieldLte
	FieldLike    = sql.FieldLike
	MatchFields  = sql.MatchFields
	HCondRef     = sql.HCondRef
	HCondTable   = sql.HCondTable
	MatchTable   = sql.MatchTable
	RawCondition = sql.RawCondition

)

func In[T any](column string, vals []T) *sql.InCond[T]       { return sql.In(column, vals) }
func NotIn[T any](column string, vals []T) *sql.NotInCond[T] { return sql.NotIn(column, vals) }

func ExistsQuery(query *Builder) sql.ExistsCond {
	return sql.ExistsCond{Query: query.inner}
}
func NotExistsQuery(query *Builder) sql.ExistsCond {
	return sql.ExistsCond{Query: query.inner, Negated: true}
}

// --- Expression helpers ---

var (
	Expr      = sql.Expr
	ExprNamed = sql.ExprNamed
	Raw       = sql.Raw
	Column    = sql.Column
	Excluded  = sql.Excluded
	Default   = sql.Default
	Func      = sql.Func
	Op        = sql.Op
	Cast      = sql.Cast

)

func SubQuery(query *Builder) sql.SQLExpr { return sql.SubQuery(query.inner) }

// --- MySQL-specific builder methods ---

func (b *Builder) ForUpdate() *Builder {
	b.inner.ForUpdate()
	return b
}

