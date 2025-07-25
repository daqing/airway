package db

import (
	"github.com/daqing/airway/lib/repo/pg"
	"github.com/daqing/airway/lib/sql"
)

func Find[T sql.Table](vals sql.H) ([]*T, error) {
	var t T

	b := sql.FindBy(t, vals)

	return pg.Find[T](pg.CurrentDB(), b)
}

func FindOne[T sql.Table](vals sql.H) (*T, error) {
	var t T

	b := sql.FindBy(t, vals)

	return pg.FindOne[T](pg.CurrentDB(), b)
}

func FindById[T sql.Table](id sql.IdType) (*T, error) {
	return FindOne[T](sql.H{
		"id": id,
	})
}
