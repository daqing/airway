package db

import (
	"github.com/daqing/airway/lib/repo/pg"
	"github.com/daqing/airway/lib/sql"
)

func Create[T sql.Table](vals sql.H) (*T, error) {
	var t T

	b := sql.Create(t, vals)

	return pg.Create[T](pg.CurrentDB(), b)
}
