package repo

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
)

func FindRow[T TableNameType](fields []string, conds []KVPair) (*T, error) {
	rows, err := Find[T](fields, conds)
	if err != nil {
		return nil, err
	}

	if len(rows) > 1 {
		return nil, ErrorCountNotMatch
	}

	if len(rows) == 0 {
		return nil, ErrorNotFound
	}

	return rows[0], nil
}

func FindAll[T TableNameType](fields []string) ([]*T, error) {
	return Find[T](fields, []KVPair{})
}

func Find[T TableNameType](fields []string, conds []KVPair) ([]*T, error) {
	return FindLimit[T](fields, conds, "", 0, 0)
}

// limit = 0 means no limit
func FindLimit[T TableNameType](fields []string, conds []KVPair, orderBy string, offset int, limit int) ([]*T, error) {
	var _t T // only used for get table name

	condQuery, values, _ := buildCondQuery(conds, 0, and_sep)

	fields = append(fields, "created_at", "updated_at")
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
	rows, err := Pool.Query(context.Background(), sql, values...)

	var ms []*T

	if err != nil {
		log.Println("[repo.Find] Conn.Query error:", err)
		return ms, err
	}

	defer rows.Close()

	for rows.Next() {
		var m = new(T)

		err := scanRows(rows, fields, m)
		if err != nil {
			log.Println("[repo.FindLimit] scanRows error:", err, "fields:", fields)
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
		camelName := ToCamel(name)
		var f = vDest.FieldByName(camelName).Addr().Interface()

		destSlice = append(destSlice, f)
	}

	if err := rows.Scan(destSlice...); err != nil {
		return err
	}

	return nil
}
