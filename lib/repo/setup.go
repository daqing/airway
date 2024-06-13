package repo

import (
	"errors"
	"log"

	"github.com/daqing/airway/lib/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var __gormDB__ *gorm.DB

var ErrNotSetup = errors.New("database is not setup yet")

func Setup() error {
	var err error

	__gormDB__, err = gorm.Open(postgres.Open(utils.GetEnvMust("AIRWAY_PG_URL")), &gorm.Config{})

	if err != nil {
		log.Printf("Failed to open database from gorm: %v", err)
		return err
	}

	return nil
}

func DB() (*gorm.DB, error) {
	if __gormDB__ == nil {
		return nil, ErrNotSetup
	}

	return __gormDB__, nil
}
