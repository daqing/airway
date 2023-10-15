package post_plugin

import "time"

type Post struct {
	Id        int64
	UserId    int64
	NodeId    int64
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "posts"

func (p Post) TableName() string { return tableName }
