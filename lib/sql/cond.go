package sql

import (
	"fmt"
	"strings"
)

type NamedArgs map[string]any

type CondBuilder interface {
	ToSQL() (string, NamedArgs)
}

type compiledCond interface {
	build(*buildState) string
}

func compileStandalone(cond CondBuilder) (string, NamedArgs) {
	state := newBuildState()
	return compileCond(cond, state), state.args
}

func compileCond(cond CondBuilder, state *buildState) string {
	if cond == nil {
		return ""
	}

	if compiled, ok := cond.(compiledCond); ok {
		return compiled.build(state)
	}

	sql, args := cond.ToSQL()
	return state.mergeExpr(sql, args)
}

type Condition struct {
	Key string
	Op  string
	Val any
}

type CompareCond struct {
	Left  any
	Op    string
	Right any
}

func (c *Condition) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c *Condition) build(state *buildState) string {
	if c == nil {
		return ""
	}

	op := strings.ToUpper(strings.TrimSpace(c.Op))
	if c.Val == nil {
		switch op {
		case "=", "IS":
			return fmt.Sprintf("%s IS NULL", c.Key)
		case "!=", "<>", "IS NOT":
			return fmt.Sprintf("%s IS NOT NULL", c.Key)
		}
	}

	return fmt.Sprintf("%s %s %s", c.Key, c.Op, renderValue(state, c.Key, c.Val))
}

func (c CompareCond) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c CompareCond) build(state *buildState) string {
	left := renderCondOperand(state, "left", c.Left)
	right := renderCondOperand(state, "right", c.Right)
	if left == "" || right == "" {
		return ""
	}

	return fmt.Sprintf("%s %s %s", left, c.Op, right)
}

func renderCondOperand(state *buildState, hint string, value any) string {
	if value == nil {
		return "NULL"
	}

	return renderValue(state, hint, value)
}

type ConditionGroupOp string

const (
	And ConditionGroupOp = "AND"
	Or  ConditionGroupOp = "OR"
)

type ConditionGroup struct {
	Left  CondBuilder
	Op    ConditionGroupOp
	Right CondBuilder
}

func (cg *ConditionGroup) ToSQL() (string, NamedArgs) {
	return compileStandalone(cg)
}

func (cg *ConditionGroup) build(state *buildState) string {
	if cg == nil {
		return ""
	}

	leftSQL := compileCond(cg.Left, state)
	rightSQL := compileCond(cg.Right, state)
	if leftSQL == "" || rightSQL == "" {
		return ""
	}

	return fmt.Sprintf("(%s %s %s)", leftSQL, cg.Op, rightSQL)
}

type EmptyCond struct{}

func (c EmptyCond) ToSQL() (string, NamedArgs) {
	return "", nil
}

func (c EmptyCond) build(*buildState) string {
	return ""
}

type AndCond struct {
	Conds []*Condition
}

func (c *AndCond) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c *AndCond) build(state *buildState) string {
	builders := make([]CondBuilder, 0, len(c.Conds))
	for _, cond := range c.Conds {
		builders = append(builders, cond)
	}

	return joinCondBuilders(builders, " AND ", false, state)
}

type OrCond struct {
	Conds []*Condition
}

func (c *OrCond) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c *OrCond) build(state *buildState) string {
	builders := make([]CondBuilder, 0, len(c.Conds))
	for _, cond := range c.Conds {
		builders = append(builders, cond)
	}

	return joinCondBuilders(builders, " OR ", false, state)
}

func joinCondBuilders(conds []CondBuilder, sep string, wrap bool, state *buildState) string {
	clauses := make([]string, 0, len(conds))
	for _, cond := range conds {
		clause := compileCond(cond, state)
		if clause == "" {
			continue
		}
		clauses = append(clauses, clause)
	}

	if len(clauses) == 0 {
		return ""
	}

	joined := strings.Join(clauses, sep)
	if wrap && len(clauses) > 1 {
		return "(" + joined + ")"
	}

	return joined
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
	return compileStandalone(c)
}

func (c *MapCond) build(state *buildState) string {
	if c == nil || len(c.Cond) == 0 {
		return ""
	}

	conds := make([]CondBuilder, 0, len(c.Cond))
	for _, key := range sortedMapKeys(c.Cond) {
		conds = append(conds, Eq(key, c.Cond[key]))
	}

	return joinCondBuilders(conds, " AND ", false, state)
}

type InCond[T any] struct {
	Column string
	Values []T
}

func (c *InCond[T]) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c *InCond[T]) build(state *buildState) string {
	if c == nil {
		return ""
	}

	if len(c.Values) == 0 {
		return "FALSE"
	}

	placeholders := make([]string, 0, len(c.Values))
	for _, v := range c.Values {
		placeholders = append(placeholders, renderValue(state, c.Column, v))
	}

	return fmt.Sprintf("%s IN (%s)", c.Column, strings.Join(placeholders, ", "))
}

type NotInCond[T any] struct {
	Column string
	Values []T
}

func (c *NotInCond[T]) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c *NotInCond[T]) build(state *buildState) string {
	if c == nil {
		return ""
	}

	if len(c.Values) == 0 {
		return "TRUE"
	}

	placeholders := make([]string, 0, len(c.Values))
	for _, v := range c.Values {
		placeholders = append(placeholders, renderValue(state, c.Column, v))
	}

	return fmt.Sprintf("%s NOT IN (%s)", c.Column, strings.Join(placeholders, ", "))
}

type BoolCond struct {
	Op    ConditionGroupOp
	Conds []CondBuilder
}

func (c BoolCond) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c BoolCond) build(state *buildState) string {
	sep := " " + string(c.Op) + " "
	return joinCondBuilders(c.Conds, sep, true, state)
}

type NotCond struct {
	Cond CondBuilder
}

func (c NotCond) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c NotCond) build(state *buildState) string {
	clause := compileCond(c.Cond, state)
	if clause == "" {
		return ""
	}

	return "NOT (" + clause + ")"
}

type BetweenCond struct {
	Column  string
	Lower   any
	Upper   any
	Negated bool
}

func (c BetweenCond) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c BetweenCond) build(state *buildState) string {
	operator := "BETWEEN"
	if c.Negated {
		operator = "NOT BETWEEN"
	}

	return fmt.Sprintf("%s %s %s AND %s", c.Column, operator, renderValue(state, c.Column+"_from", c.Lower), renderValue(state, c.Column+"_to", c.Upper))
}

type NullCond struct {
	Column  string
	Negated bool
}

func (c NullCond) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c NullCond) build(*buildState) string {
	if c.Negated {
		return c.Column + " IS NOT NULL"
	}

	return c.Column + " IS NULL"
}

type ExistsCond struct {
	Query   *Builder
	Negated bool
}

func (c ExistsCond) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c ExistsCond) build(state *buildState) string {
	if c.Query == nil {
		return ""
	}

	querySQL := SubQuery(c.Query).buildExpr(state)
	if c.Negated {
		return "NOT EXISTS " + querySQL
	}

	return "EXISTS " + querySQL
}

type rawCond struct {
	SQL  string
	Args NamedArgs
}

func (c rawCond) ToSQL() (string, NamedArgs) {
	return compileStandalone(c)
}

func (c rawCond) build(state *buildState) string {
	return state.mergeExpr(c.SQL, c.Args)
}
