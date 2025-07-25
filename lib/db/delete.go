package db

import (
	"github.com/daqing/airway/lib/repo/pg"
	"github.com/daqing/airway/lib/sql"
)

func DeleteById[T sql.Table](id sql.IdType) error {
	return Delete[T](sql.H{"id": id})
}

func Delete[T sql.Table](vals sql.H) error {
	var t T

	b := sql.Delete().From(t.TableName()).Where(&sql.MapCond{Cond: vals})

	return pg.Delete(pg.CurrentDB(), b)
}
