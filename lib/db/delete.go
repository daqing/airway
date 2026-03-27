package db

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
)

func DeleteById[T sql.Table](id sql.IdType) error {
	return Delete[T](sql.H{"id": id})
}

func Delete[T sql.Table](vals sql.H) error {
	var t T

	b := sql.DeleteFrom(sql.TableFor(t)).Where(sql.MatchTable(t, vals))

	return repo.Delete(repo.CurrentDB(), b)
}
