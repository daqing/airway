package sql

import "strings"

func Eq(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "=",
		Val: val,
	}
}

func NotEq(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "<>",
		Val: val,
	}
}

func Gt(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  ">",
		Val: val,
	}
}

func Gte(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  ">=",
		Val: val,
	}
}

func Lt(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "<",
		Val: val,
	}
}

func Lte(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "<=",
		Val: val,
	}
}

func Like(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "LIKE",
		Val: val,
	}
}

func NotLike(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "NOT LIKE",
		Val: val,
	}
}

func ILike(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "ILIKE",
		Val: val,
	}
}

func NotILike(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "NOT ILIKE",
		Val: val,
	}
}

func HCond(cond H) *MapCond {
	return &MapCond{
		Cond: cond,
	}
}

func In[T any](column string, vals []T) *InCond[T] {
	return &InCond[T]{
		Column: column,
		Values: vals,
	}
}

func NotIn[T any](column string, vals []T) *NotInCond[T] {
	return &NotInCond[T]{
		Column: column,
		Values: vals,
	}
}

func AllOf(conds ...CondBuilder) BoolCond {
	return BoolCond{Op: And, Conds: conds}
}

func AnyOf(conds ...CondBuilder) BoolCond {
	return BoolCond{Op: Or, Conds: conds}
}

func Not(cond CondBuilder) NotCond {
	return NotCond{Cond: cond}
}

func IsNull(column string) NullCond {
	return NullCond{Column: column}
}

func IsNotNull(column string) NullCond {
	return NullCond{Column: column, Negated: true}
}

func Between(column string, lower, upper any) BetweenCond {
	return BetweenCond{Column: column, Lower: lower, Upper: upper}
}

func NotBetween(column string, lower, upper any) BetweenCond {
	return BetweenCond{Column: column, Lower: lower, Upper: upper, Negated: true}
}

func ExistsQuery(query *Builder) ExistsCond {
	return ExistsCond{Query: query}
}

func NotExistsQuery(query *Builder) ExistsCond {
	return ExistsCond{Query: query, Negated: true}
}

func RawCondition(sql string, args NamedArgs) CondBuilder {
	return rawCond{SQL: sql, Args: args}
}

func Compare(left any, op string, right any) CompareCond {
	return CompareCond{Left: left, Op: op, Right: right}
}

func FieldEq(column FieldName, val any) CompareCond {
	return Compare(column.Ref(), "=", val)
}

func EqRef(column FieldName, val any) CompareCond {
	return FieldEq(column, val)
}

func FieldNotEq(column FieldName, val any) CompareCond {
	return Compare(column.Ref(), "<>", val)
}

func NotEqRef(column FieldName, val any) CompareCond {
	return FieldNotEq(column, val)
}

func FieldGt(column FieldName, val any) CompareCond {
	return Compare(column.Ref(), ">", val)
}

func GtRef(column FieldName, val any) CompareCond {
	return FieldGt(column, val)
}

func FieldGte(column FieldName, val any) CompareCond {
	return Compare(column.Ref(), ">=", val)
}

func GteRef(column FieldName, val any) CompareCond {
	return FieldGte(column, val)
}

func FieldLt(column FieldName, val any) CompareCond {
	return Compare(column.Ref(), "<", val)
}

func LtRef(column FieldName, val any) CompareCond {
	return FieldLt(column, val)
}

func FieldLte(column FieldName, val any) CompareCond {
	return Compare(column.Ref(), "<=", val)
}

func LteRef(column FieldName, val any) CompareCond {
	return FieldLte(column, val)
}

func FieldLike(column FieldName, val any) CompareCond {
	return Compare(column.Ref(), "LIKE", val)
}

func LikeRef(column FieldName, val any) CompareCond {
	return FieldLike(column, val)
}

func FieldILike(column FieldName, val any) CompareCond {
	return Compare(column.Ref(), "ILIKE", val)
}

func ILikeRef(column FieldName, val any) CompareCond {
	return FieldILike(column, val)
}

func MatchFields(table TableName, cond H) CondBuilder {
	if len(cond) == 0 {
		return EmptyCond{}
	}

	conds := make([]CondBuilder, 0, len(cond))
	for _, key := range sortedMapKeys(cond) {
		conds = append(conds, FieldEq(table.Field(key), cond[key]))
	}

	return AllOf(conds...)
}

func HCondRef(table TableName, cond H) CondBuilder {
	return MatchFields(table, cond)
}

func HCondTable(t Table, cond H) CondBuilder {
	return MatchTable(t, cond)
}

func MatchTable(t Table, cond H) CondBuilder {
	return MatchFields(TableFor(t), cond)
}

func ArrayContains(left, right any) CompareCond {
	return Compare(left, "@>", right)
}

func ArrayContainedBy(left, right any) CompareCond {
	return Compare(left, "<@", right)
}

func ArrayOverlap(left, right any) CompareCond {
	return Compare(left, "&&", right)
}

func JSONGet(left any, key any) SQLExpr {
	return Op(left, "->", key)
}

func JSONGetText(left any, key any) SQLExpr {
	return Op(left, "->>", key)
}

func JSONPath(left any, path ...string) SQLExpr {
	return Op(left, "#>", Cast(Array(stringsToAny(path)...), "text[]"))
}

func JSONPathText(left any, path ...string) SQLExpr {
	return Op(left, "#>>", Cast(Array(stringsToAny(path)...), "text[]"))
}

func JSONContains(left, right any) CompareCond {
	return Compare(left, "@>", right)
}

func JSONContainedBy(left, right any) CompareCond {
	return Compare(left, "<@", right)
}

func JSONHasKey(left any, key string) CompareCond {
	return Compare(left, "?", key)
}

func JSONHasAnyKeys(left any, keys ...string) CompareCond {
	return Compare(left, "?|", Array(stringsToAny(keys)...))
}

func JSONHasAllKeys(left any, keys ...string) CompareCond {
	return Compare(left, "?&", Array(stringsToAny(keys)...))
}

func stringsToAny(values []string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}

	return result
}
