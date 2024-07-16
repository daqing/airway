package models

import (
	"time"
)

type MediaFile struct {
	BaseModel

	UserId    int64
	Filename  string
	Mime      string
	Bytes     int64
	ExpiredAt time.Time
}

func (m MediaFile) TableName() string { return "media_files" }
