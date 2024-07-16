package models

import (
	"time"
)

type Checkin struct {
	BaseModel

	UserId IdType

	Year  int
	Month time.Month
	Day   int

	Acc int // 连续签到次数
}

func (c Checkin) TableName() string { return "checkins" }
