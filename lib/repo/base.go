package repo

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const eq_op = "="
const in_op = "IN"
const or_op = "OR"

var ErrorNotFound = errors.New("record_not_found")
var ErrorCountNotMatch = errors.New("count_not_match")

var NeverExpires = time.Now().AddDate(100, 0, 0)

const InvalidCount = -1

type Separator string

const and_sep Separator = " AND "
const or_sep Separator = " OR "
const comma_sep Separator = ", "

type KVPair interface {
	Key() string
	Value() any
	Operator() string
}

type Attribute struct {
	KeyField   string
	ValueField any
}

func KV(key string, value any) *Attribute {
	return &Attribute{key, value}
}

func (attr *Attribute) Key() string {
	return attr.KeyField
}

func (attr *Attribute) Value() any {
	return attr.ValueField
}

func (attr *Attribute) Operator() string {
	return eq_op
}

const EMPTY_STRING = ""

type OrQuery struct {
	Pairs []KVPair
}

func (or *OrQuery) Key() string {
	return EMPTY_STRING
}

func (or *OrQuery) Value() any {
	var result []any

	for _, pair := range or.Pairs {
		result = append(result, pair.Value())
	}

	return result
}

func (or *OrQuery) Operator() string {
	return or_op
}

func OR(pairs ...KVPair) *OrQuery {
	return &OrQuery{Pairs: pairs}
}

type InQuery[T any] struct {
	Field  string
	Values []T
}

func In[T any](field string, values []T) *InQuery[T] {
	return &InQuery[T]{field, values}
}

func (in *InQuery[T]) Key() string {
	return in.Field
}

func (in *InQuery[T]) Value() any {
	var result []string

	for _, v := range in.Values {
		result = append(result, fmt.Sprintf("%v", v))
	}

	return strings.Join(result, string(comma_sep))
}

func (in *InQuery[T]) Operator() string { return in_op }

// return the next available dollar sign
//
// e.g buildCondQuery([]KVPair{KV("foo", "bar")}, 0, AND) will return dollar value as 2
func buildCondQuery(conds []KVPair, start int, sep Separator) (condQuery string, values []any, dollar int) {
	if len(conds) == 0 {
		return "1=1", nil, 0
	}

	dollar = start
	var condString = []string{}

	for _, cond := range conds {

		var part string

		switch cond.Operator() {
		case in_op:
			dollar++
			part = fmt.Sprintf("%s IN ($%d)", cond.Key(), dollar)

			values = append(values, cond.Value())
		case or_op:
			if or, ok := cond.(*OrQuery); ok {
				var subParts []string

				for _, subCond := range or.Pairs {
					dollar++

					subParts = append(
						subParts,
						fmt.Sprintf("%s %s $%d", subCond.Key(), subCond.Operator(), dollar),
					)

					values = append(values, subCond.Value())
				}

				part = strings.Join(subParts, string(or_sep))
			} else {
				panic("invalid OR query")
			}

		default:
			dollar++
			part = fmt.Sprintf("%s %s $%d", cond.Key(), cond.Operator(), dollar)

			values = append(values, cond.Value())
		}

		condString = append(condString, part)
	}

	condQuery = strings.Join(condString, string(sep))
	dollar++

	return
}

// Polymorphic model
type PolyModel interface {
	PolyId() int64
	PolyType() string
}

type PriceCent int64

func ToCent(price float64) PriceCent {
	return PriceCent(price * 100)
}

func (c PriceCent) Yuan() string {
	return fmt.Sprintf("%.2f", float64(c/100.0))
}
