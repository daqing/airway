package media_api

import (
	"time"
)

type MediaFile struct {
	Id int64

	UserId    int64
	Filename  string
	Mime      string
	Bytes     int64
	ExpiredAt time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "media_files"

func (m MediaFile) TableName() string { return tableName }
