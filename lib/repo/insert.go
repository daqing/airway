package repo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/daqing/airway/lib/utils"
)

func Insert[T TableNameType](attributes []KeyValueField) (*T, error) {
	return InsertSkipExists[T](attributes, false)
}

func InsertSkipExists[T TableNameType](attributes []KeyValueField, skipExists bool) (*T, error) {
	if skipExists {
		ex, err := Exists[T](attributes)
		if err != nil {
			return nil, err
		}

		if ex {
			return nil, nil
		}
	}

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
	row := Pool.QueryRow(context.Background(), sql, values...)

	var id int64

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
