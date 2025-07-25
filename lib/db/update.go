package db

import (
	"github.com/daqing/airway/lib/repo/pg"
	"github.com/daqing/airway/lib/sql"
)

func Update[T sql.Table](vals sql.H, cond sql.CondBuilder) error {
	var t T

	b := sql.Update(t.TableName()).Set(vals).Where(cond)

	return pg.Update(pg.CurrentDB(), b)
}

func UpdateById[T sql.Table](id sql.IdType, vals sql.H) error {
	return Update[T](vals, sql.Eq("id", id))
}

func UpdateAll[T sql.Table](vals sql.H) error {
	var t T

	b := sql.UpdateAll(t, vals)

	return pg.Update(pg.CurrentDB(), b)
}
