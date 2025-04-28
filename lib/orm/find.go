package orm

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func FindOne[T Table](db *DB, fields []string, cond CondBuilder) (*T, error) {
	rows, err := Find[T](db, fields, cond)
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

func FindAll[T Table](db *DB, fields []string) ([]*T, error) {
	return Find[T](db, fields, EmptyCond{})
}

func Find[T Table](db *DB, fields []string, cond CondBuilder) ([]*T, error) {
	return FindLimit[T](db, fields, cond, "", 0, 0)
}

// limit = 0 means no limit
func FindLimit[T Table](db *DB, fields []string, cond CondBuilder, orderBy string, offset int, limit int) ([]*T, error) {
	var t T

	where, vals := whereQuery(cond)
	var field string
	if len(fields) == 0 {
		field = "*"
	} else {
		field = strings.Join(fields, ",")
	}

	sql := "SELECT " + field + " FROM " + t.TableName()
	if len(cond.Cond()) > 0 {
		sql += " WHERE " + where
	}

	if orderBy != EMPTY_STRING {
		sql += " ORDER BY " + orderBy
	}

	if limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", limit)
	}

	if offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", offset)
	}

	fmt.Println(sql)

	var records = []*T{}

	rows, err := db.pool.Query(context.Background(), sql, vals...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var record T
		record, err := pgx.RowToStructByName[T](rows)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	return records, nil
}
