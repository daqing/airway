package app

import (
	"github.com/daqing/airway/app/models"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&models.Action{})
	db.AutoMigrate(&models.Checkin{})
	db.AutoMigrate(&models.Comment{})
	db.AutoMigrate(&models.MediaFile{})
	db.AutoMigrate(&models.Membership{})
	db.AutoMigrate(&models.Menu{})
	db.AutoMigrate(&models.Node{})
	db.AutoMigrate(&models.Payment{})
	db.AutoMigrate(&models.Point{})
	db.AutoMigrate(&models.Post{})
	db.AutoMigrate(&models.Setting{})
	db.AutoMigrate(&models.Tag{})
	db.AutoMigrate(&models.User{})
}
