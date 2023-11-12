package post_api

import "time"

type Post struct {
	Id         int64
	UserId     int64
	NodeId     int64
	Title      string
	CustomPath string
	Cat        string
	Content    string
	Fee        int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const tableName = "posts"

func (p Post) TableName() string { return tableName }

const polyType = "post"

func (p *Post) PolyId() int64    { return p.Id }
func (p *Post) PolyType() string { return polyType }
