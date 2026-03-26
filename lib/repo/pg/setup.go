package pg

import (
	"errors"
)

var __DB__ *DB

var ErrNotSetup = errors.New("database is not setup yet")

func Setup(dsn string) (*DB, error) {
	return SetupWithDriver("", dsn)
}

func SetupWithDriver(driverName string, dsn string) (*DB, error) {
	db, err := NewDBWithDriver(driverName, dsn)
	if err != nil {
		return nil, err
	}

	if __DB__ != nil {
		_ = __DB__.Close()
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
