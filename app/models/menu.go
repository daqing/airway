package models

import (
	"gorm.io/gorm"
)

type Menu struct {
	gorm.Model

	Name  string
	URL   string
	Place string
}

func (m Menu) TableName() string { return "menus" }
