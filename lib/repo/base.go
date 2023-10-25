package repo

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const EQ = "="
const IN = "IN"

// var ErrorNotFound = errors.New("record_not_found")
var ErrorCountNotMatch = errors.New("count_not_match")

var NeverExpires = time.Now().AddDate(100, 0, 0)

const InvalidCount = -1

type Separator string

const AND Separator = " AND "
const OR Separator = " OR "
const COMMA Separator = " , "

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
	return EQ
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

	return strings.Join(result, ",")
}

func (in *InQuery[T]) Operator() string { return IN }

func buildCondQuery(conds []KVPair, start int, sep Separator) (condQuery string, values []any, dollar int) {
	if len(conds) == 0 {
		return "1=1", nil, 0
	}

	dollar = start
	var condString = []string{}

	for _, cond := range conds {
		dollar += 1

		var part string

		switch cond.Operator() {
		case IN:
			part = fmt.Sprintf("%s IN ($%d)", cond.Key(), dollar)
		default:
			part = fmt.Sprintf("%s %s $%d", cond.Key(), cond.Operator(), dollar)
		}

		condString = append(condString, part)
		values = append(values, cond.Value())
	}

	condQuery = strings.Join(condString, string(sep))

	dollar++

	return
}
