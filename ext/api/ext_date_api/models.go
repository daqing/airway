package ext_date_api

import (
	"time"
)

type Date struct {
	Id int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "dates"

func (m Date) TableName() string { return tableName }
