package comment_plugin

import "time"

type Comment struct {
	Id int64

	TargetId   int64
	TargetType string
	Content    string

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "comments"

func (c Comment) TableName() string { return tableName }
