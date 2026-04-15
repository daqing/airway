// Package cond provides generic SQL condition constructors.
//
// Use cond.Eq, cond.Like, cond.AllOf, etc. when building WHERE clauses
// so query conditions are not tied to a specific dialect package.
package cond

import sql "github.com/daqing/airway/lib/sql"

// Condition types
type (
	CondBuilder      = sql.CondBuilder
	Condition        = sql.Condition
	CompareCond      = sql.CompareCond
	ConditionGroup   = sql.ConditionGroup
	ConditionGroupOp = sql.ConditionGroupOp
	EmptyCond        = sql.EmptyCond
	AndCond          = sql.AndCond
	OrCond           = sql.OrCond
	MapCond          = sql.MapCond
	BoolCond         = sql.BoolCond
	NotCond          = sql.NotCond
	BetweenCond      = sql.BetweenCond
	NullCond         = sql.NullCond
	ExistsCond       = sql.ExistsCond
	InCond[T any]    = sql.InCond[T]
	NotInCond[T any] = sql.NotInCond[T]
)

// Comparison constructors
var (
	Eq           = sql.Eq
	NotEq        = sql.NotEq
	Gt           = sql.Gt
	Gte          = sql.Gte
	Lt           = sql.Lt
	Lte          = sql.Lte
	Like         = sql.Like
	NotLike      = sql.NotLike
	ILike        = sql.ILike
	NotILike     = sql.NotILike
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
	FieldILike   = sql.FieldILike
	MatchFields  = sql.MatchFields
	HCondRef     = sql.HCondRef
	HCondTable   = sql.HCondTable
	MatchTable   = sql.MatchTable
	RawCondition = sql.RawCondition
)

// In creates an IN condition.
func In[T any](column string, vals []T) *sql.InCond[T] { return sql.In(column, vals) }

// NotIn creates a NOT IN condition.
func NotIn[T any](column string, vals []T) *sql.NotInCond[T] { return sql.NotIn(column, vals) }

// ExistsQuery creates an EXISTS condition.
func ExistsQuery(query *sql.Builder) sql.ExistsCond { return sql.ExistsQuery(query) }

// NotExistsQuery creates a NOT EXISTS condition.
func NotExistsQuery(query *sql.Builder) sql.ExistsCond { return sql.NotExistsQuery(query) }
