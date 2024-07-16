package sql_orm

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const op_eq = "="
const op_in = "IN"
const op_or = "OR"

var ErrorNotFound = errors.New("record_not_found")
var ErrorCountNotMatch = errors.New("count_not_match")

var NeverExpires = time.Now().AddDate(100, 0, 0)

const InvalidCount = -1

type Separator string

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
	return op_eq
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
	return op_or
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

func (in *InQuery[T]) Operator() string { return op_in }

func buildCondQuery(kvpairs []KVPair) map[string]any {
	result := make(map[string]any)

	for _, kv := range kvpairs {
		if kv.Operator() == op_eq {
			result[kv.Key()] = kv.Value()
		}
	}

	return result
}

// type Model struct {
// 	ID        IdType `gorm:"primarykey"`
// 	CreatedAt time.Time
// 	UpdatedAt time.Time
// }
