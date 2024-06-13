package models

import (
	"time"

	"gorm.io/gorm"
)

type Checkin struct {
	gorm.Model

	UserId uint

	Year  int
	Month time.Month
	Day   int

	Acc int // 连续签到次数
}

func (c Checkin) TableName() string { return "checkins" }
