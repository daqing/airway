package app

import (
	"github.com/daqing/airway/app/models"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&models.User{})
}
