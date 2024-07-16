package models

type Comment struct {
	BaseModel

	UserId IdType

	TargetId   IdType
	TargetType string
	Content    string
}

func (c Comment) TableName() string { return "comments" }
