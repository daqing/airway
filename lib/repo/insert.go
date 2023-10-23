package repo

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
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

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id, created_at, updated_at", t.TableName(), fieldQuery, dollarQuery)
	row := Pool.QueryRow(context.Background(), sql, values...)

	var id int64
	var createdAt time.Time
	var updatedAt time.Time

	err := row.Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	if id == 0 {
		return nil, errors.New("ID is zero")
	}

	attributes = append(
		attributes,
		NewKV("id", id),
		NewKV("created_at", createdAt),
		NewKV("updated_at", updatedAt),
	)

	assignAttributes(&t, attributes)

	return &t, err
}

func assignAttributes(dest any, attributes []KeyValueField) {
	vDest := reflect.ValueOf(dest).Elem()

	for _, attr := range attributes {
		camelName := ToCamel(attr.KeyField())
		var f = vDest.FieldByName(camelName)

		if f.CanSet() {
			f.Set(reflect.ValueOf(attr.ValueField()))
		}
	}
}
