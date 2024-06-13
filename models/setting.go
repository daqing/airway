package models

import (
	"gorm.io/gorm"
)

type Setting struct {
	gorm.Model

	Key string
	Val string
}

func (s Setting) TableName() string { return "settings" }
