package models

// 用户积分
type Point struct {
	BaseModel

	UserId IdType
	Count  int
}

func (p Point) TableName() string { return "points" }
