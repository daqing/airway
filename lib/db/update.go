package db

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
)

func Update[T sql.Table](vals sql.H, cond sql.CondBuilder) error {
	var t T

	b := sql.UpdateTable(sql.TableFor(t)).Set(vals).Where(cond)

	return repo.Update(repo.CurrentDB(), b)
}

func UpdateById[T sql.Table](id sql.IdType, vals sql.H) error {
	var t T
	return Update[T](vals, sql.FieldEq(sql.FieldFor(t, "id"), id))
}

func UpdateAll[T sql.Table](vals sql.H) error {
	var t T

	b := sql.UpdateAll(t, vals)

	return repo.Update(repo.CurrentDB(), b)
}
