package point_api

import (
	"time"
)

// 用户积分
type Point struct {
	Id int64

	UserId int64
	Count  int

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "points"

func (m Point) TableName() string { return tableName }
