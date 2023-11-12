package checkin_api

import "time"

type Checkin struct {
	Id int64

	UserId int64

	Year  int
	Month time.Month
	Day   int

	Acc int // 连续签到次数

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "checkin"

func (c Checkin) TableName() string { return tableName }
