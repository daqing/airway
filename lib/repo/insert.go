package repo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/daqing/airway/lib/utils"
)

type KeyValueField interface {
	KeyField() string
	ValueField() any
}

type Attribute struct {
	Key   string
	Value any
}

func NewKV(key string, value any) *Attribute {
	return &Attribute{key, value}
}

func (attr *Attribute) KeyField() string {
	return attr.Key
}

func (attr *Attribute) ValueField() any {
	return attr.Value
}

func Insert[T TableNameType, Id int | int64](attributes []KeyValueField) (*T, error) {
	var fields []string

	var dollars []string
	var values []any

	n := 0
	for _, field := range attributes {
		fields = append(fields, field.KeyField())

		n++
		dollars = append(dollars, fmt.Sprintf("$%d", n))
		values = append(values, field.ValueField())
	}

	fieldQuery := strings.Join(fields, ",")
	dollarQuery := strings.Join(dollars, ",")

	var t T

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id", t.TableName(), fieldQuery, dollarQuery)
	row := Conn.QueryRow(context.Background(), sql, values...)

	var id Id

	err := row.Scan(&id)
	if err != nil {
		return nil, err
	}

	if id == 0 {
		return nil, errors.New("ID is zero")
	}

	attributes = append(attributes, NewKV("id", id))
	assignAttributes(&t, attributes)

	return &t, err
}

func assignAttributes(dest any, attributes []KeyValueField) {
	vDest := reflect.ValueOf(dest).Elem()

	for _, attr := range attributes {
		camelName := utils.ToCamel(attr.KeyField())
		var f = vDest.FieldByName(camelName)

		if f.CanSet() {
			f.Set(reflect.ValueOf(attr.ValueField()))
		}
	}
}
