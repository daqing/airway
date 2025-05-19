package sql

import (
	"fmt"
	"strings"
)

type NamedArgs map[string]any

type CondBuilder interface {
	ToSQL() (string, NamedArgs)
}

type Condition struct {
	Key string
	Op  string
	Val any
}

func (c *Condition) ToSQL() (string, NamedArgs) {
	var args NamedArgs = map[string]any{
		c.Key: c.Val,
	}

	return fmt.Sprintf("%s %s @%s", c.Key, c.Op, c.Key), args
}

type ConditionGroupOp string

const (
	And ConditionGroupOp = "AND"
	Or  ConditionGroupOp = "OR"
)

type ConditionGroup struct {
	Left  *Condition
	Op    ConditionGroupOp
	Right *Condition
}

func (cg *ConditionGroup) ToSQL() (string, NamedArgs) {
	if cg.Left == nil || cg.Right == nil {
		return "", nil
	}

	leftSQL, leftArgs := cg.Left.ToSQL()
	rightSQL, rightArgs := cg.Right.ToSQL()

	sql := fmt.Sprintf("(%s %s %s)", leftSQL, cg.Op, rightSQL)

	return sql, mergeMap(leftArgs, rightArgs)
}

type EmptyCond struct{}

func (c EmptyCond) ToSQL() (string, NamedArgs) {
	return "", nil
}

type AndCond struct {
	Conds []*Condition
}

func (c *AndCond) ToSQL() (string, NamedArgs) {
	return joinConds(c.Conds, " AND ")
}

type OrCond struct {
	Conds []*Condition
}

func (c *OrCond) ToSQL() (string, NamedArgs) {
	return joinConds(c.Conds, " OR ")
}

func joinConds(conds []*Condition, sep string) (string, NamedArgs) {
	if len(conds) == 0 {
		return "", nil
	}

	var clauses []string
	var result NamedArgs = make(map[string]any)

	for _, cond := range conds {
		clauses = append(clauses, fmt.Sprintf("%s %s @%s", cond.Key, cond.Op, cond.Key))
		result[cond.Key] = cond.Val
	}

	sql := strings.Join(clauses, sep)

	return sql, result
}

func mergeMap(m1, m2 NamedArgs) NamedArgs {
	merged := make(NamedArgs)

	for k, v := range m1 {
		merged[k] = v
	}

	for k, v := range m2 {
		merged[k] = v
	}

	return merged
}

type MapCond struct {
	Cond H
}

func (c *MapCond) ToSQL() (string, NamedArgs) {
	var and AndCond

	for k, v := range c.Cond {
		and.Conds = append(and.Conds, Eq(k, v))
	}

	return and.ToSQL()
}

type InCond[T any] struct {
	Column string
	Values []T
}

func (c *InCond[T]) ToSQL() (string, NamedArgs) {
	if len(c.Values) == 0 {
		return "", nil
	}

	var args NamedArgs = make(map[string]any)
	var placeholders []string

	for i, v := range c.Values {
		key := fmt.Sprintf("%s_%d", c.Column, i)
		args[key] = v
		placeholders = append(placeholders, fmt.Sprintf("@%s", key))
	}

	sql := fmt.Sprintf("%s IN (%s)", c.Column, strings.Join(placeholders, ", "))

	return sql, args
}
