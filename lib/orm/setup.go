package orm

import (
	"errors"
	"log"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var __gormDB__ *gorm.DB

var ErrNotSetup = errors.New("database is not setup yet")

func Setup() error {
	// don't setup if no AIRWAY_PG_URL is set
	if _, err := utils.GetEnv("AIRWAY_PG_URL"); err != nil {
		return nil
	}

	var err error

	__gormDB__, err = gorm.Open(postgres.Open(utils.GetEnvMust("AIRWAY_PG_URL")), &gorm.Config{})

	if err != nil {
		log.Printf("Failed to open database from gorm: %v", err)
		return err
	}

	autoMigrate()

	return nil
}

func DB() *gorm.DB {
	if __gormDB__ == nil {
		panic(ErrNotSetup)
	}

	return __gormDB__
}

func autoMigrate() {
	db := __gormDB__

	db.AutoMigrate(&models.User{})
}
