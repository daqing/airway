package models

import (
	"time"

	"gorm.io/gorm"
)

type MediaFile struct {
	gorm.Model

	UserId    int64
	Filename  string
	Mime      string
	Bytes     int64
	ExpiredAt time.Time
}

func (m MediaFile) TableName() string { return "media_files" }
