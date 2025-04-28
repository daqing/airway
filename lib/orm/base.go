package orm

import (
	"errors"
	"time"
)

var ErrorNotFound = errors.New("record_not_found")
var ErrorCountNotMatch = errors.New("count_not_match")

var NeverExpires = time.Now().AddDate(100, 0, 0)

const InvalidCount = -1

type Separator string

type CondBuilder interface {
	Cond() map[string]any
}

type kv struct {
	KeyField   string
	ValueField any
}

func (kv *kv) Cond() map[string]any {
	return map[string]any{kv.KeyField: kv.ValueField}
}

func Eq(key string, value any) *kv {
	return &kv{key, value}
}

const EMPTY_STRING = ""

type EmptyCond struct{}

func (EmptyCond) Cond() map[string]any {
	return map[string]any{}
}

type Fields struct {
	Fields []*kv
}

func (f *Fields) Keys() []string {
	var keys []string
	for _, kv := range f.Fields {
		keys = append(keys, kv.KeyField)
	}
	return keys
}

func (f *Fields) Values() []any {
	var values []any
	for _, kv := range f.Fields {
		values = append(values, kv.ValueField)
	}
	return values
}

func (f *Fields) Cond() map[string]any {
	result := make(map[string]any)

	for _, kv := range f.Fields {
		result[kv.KeyField] = kv.ValueField
	}

	return result
}

func MultiFields(fields ...*kv) *Fields {
	return &Fields{fields}
}

func (f *Fields) ToMap() map[string]any {
	return f.Cond()
}
