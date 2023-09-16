package repo

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/daqing/airway/lib/utils"
	"github.com/jackc/pgx/v5"
)

type QueryCond struct {
	Key   string
	Value any
}

func (c *QueryCond) KeyField() string {
	return c.Key
}

func (c *QueryCond) ValueField() any {
	return c.Value
}

func NewCond(key string, value any) *QueryCond {
	return &QueryCond{key, value}
}

func Find[T TableNameType](fields []string, conds []KeyValueField) ([]*T, error) {
	var condString = []string{}
	var values = []any{}
	var dollar int

	var _t T // only used for get table name

	for _, cond := range conds {
		dollar += 1

		part := fmt.Sprintf("%s = $%d", cond.KeyField(), dollar)

		condString = append(condString, part)
		values = append(values, cond.ValueField())
	}

	var condQuery = strings.Join(condString, " AND ")
	var fieldsQuery = strings.Join(fields, ", ")

	sql := fmt.Sprintf("SELECT %s FROM %s WHERE %s", fieldsQuery, _t.TableName(), condQuery)
	fmt.Printf("[repo.Find] SQL: %s, values: %+v\n", sql, values)

	rows, err := Conn.Query(context.Background(), sql, values...)

	var ms []*T

	if err != nil {
		fmt.Println("[repo.Find] Conn.Query error:", err)
		return ms, err
	}

	defer rows.Close()

	for rows.Next() {
		var m = new(T)

		err := scanRows(rows, fields, m)
		if err != nil {
			fmt.Println("[repo.Find] scanRows error:", err, "fields:", fields)
			return ms, err
		}

		ms = append(ms, m)
	}

	return ms, nil
}

func scanRows(rows pgx.Rows, fields []string, dest any) error {
	vDest := reflect.ValueOf(dest).Elem()
	destSlice := make([]interface{}, 0)

	for _, name := range fields {
		camelName := utils.ToCamel(name)
		var f = vDest.FieldByName(camelName).Addr().Interface()

		destSlice = append(destSlice, f)
	}

	if err := rows.Scan(destSlice...); err != nil {
		return err
	}

	return nil
}
