package repo

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/daqing/airway/lib/utils"
	"github.com/jackc/pgx/v5"
)

func Find[T TableNameType](fields []string, conds []KeyValueField) ([]*T, error) {
	return FindLimit[T](fields, conds, "", 0, 0)
}

// limit = 0 means no limit
func FindLimit[T TableNameType](fields []string, conds []KeyValueField, orderBy string, offset int, limit int) ([]*T, error) {
	var _t T // only used for get table name

	condQuery, values := buildCondQuery(conds)

	fieldsQuery := strings.Join(fields, ", ")

	var sql string
	if limit > 0 {
		sql = fmt.Sprintf("SELECT %s FROM %s WHERE %s order by %s offset %d limit %d", fieldsQuery, _t.TableName(), condQuery, orderBy, offset, limit)
	} else if len(orderBy) > 0 {
		sql = fmt.Sprintf("SELECT %s FROM %s WHERE %s ORDER BY %s", fieldsQuery, _t.TableName(), condQuery, orderBy)
	} else {
		sql = fmt.Sprintf("SELECT %s FROM %s WHERE %s", fieldsQuery, _t.TableName(), condQuery)
	}

	return execSQL[T](sql, fields, values)

}

func execSQL[T TableNameType](sql string, fields []string, values []any) ([]*T, error) {
	fmt.Printf("SQL: %s, values: %+v\n", sql, values)

	rows, err := Pool.Query(context.Background(), sql, values...)

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
			fmt.Println("[repo.FindLimit] scanRows error:", err, "fields:", fields)
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
