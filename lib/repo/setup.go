package repo

import (
	"errors"
)

var __DB__ *DB

var ErrNotSetup = errors.New("database is not setup yet")

func Setup(dsn string) (*DB, error) {
	return SetupDB(dsn)
}

func SetupWithDriver(driverName string, dsn string) (*DB, error) {
	return SetupDBWithDriver(driverName, dsn)
}

func SetupDB(dsn string) (*DB, error) {
	return SetupDBWithDriver("", dsn)
}

func SetupDBWithDriver(driverName string, dsn string) (*DB, error) {
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

func CurrentDBOK() (*DB, bool) {
	if __DB__ == nil {
		return nil, false
	}

	return __DB__, true
}
