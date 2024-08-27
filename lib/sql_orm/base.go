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

type CondBuilder interface {
	Cond() string
}

type kv struct {
	KeyField   string
	ValueField any
}

func (kv *kv) Cond() string {
	return fmt.Sprintf("%s = %v", kv.KeyField, kv.ValueField)
}

func Eq(key string, value any) *kv {
	return &kv{key, value}
}

const EMPTY_STRING = ""
const DEFAULT_COND = "1=1"

type EmptyCond struct{}

func (EmptyCond) Cond() string {
	return DEFAULT_COND
}

type Fields struct {
	Fields []*kv
}

func (f *Fields) Cond() string {
	conditions := make([]string, len(f.Fields))

	for i, kv := range f.Fields {
		conditions[i] = kv.Cond()
	}

	return strings.Join(conditions, ", ")
}

func MultiFields(fields ...*kv) *Fields {
	return &Fields{fields}
}

func (f *Fields) ToMap() map[string]any {
	row := make(map[string]any)

	for _, kv := range f.Fields {
		row[kv.KeyField] = kv.ValueField
	}

	return row
}
