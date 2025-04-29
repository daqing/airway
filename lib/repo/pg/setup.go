package pg

import (
	"errors"
)

var __DB__ *DB

var ErrNotSetup = errors.New("database is not setup yet")

func Setup(dsn string) (*DB, error) {
	db, err := NewDB(dsn)
	if err != nil {
		return nil, err
	}

	__DB__ = db
	return db, nil
}

func CurrentDB() *DB {
	if __DB__ == nil {
		panic(ErrNotSetup)
	}

	return __DB__
}
