package models

import "gorm.io/gorm"

// 用户积分
type Point struct {
	gorm.Model

	UserId uint
	Count  int
}

func (p Point) TableName() string { return "points" }
