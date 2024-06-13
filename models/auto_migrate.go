package models

import "gorm.io/gorm"

func AutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&Action{})
	db.AutoMigrate(&Checkin{})
	db.AutoMigrate(&Comment{})
	db.AutoMigrate(&MediaFile{})
	db.AutoMigrate(&Membership{})
	db.AutoMigrate(&Menu{})
	db.AutoMigrate(&Node{})
	db.AutoMigrate(&Payment{})
	db.AutoMigrate(&Point{})
	db.AutoMigrate(&Post{})
	db.AutoMigrate(&Setting{})
	db.AutoMigrate(&Tag{})
	db.AutoMigrate(&User{})
}
