package hello_plugin

import "time"

type Hello struct {
	Id        int64
	Name      string
	Age       int
	CreatedAt time.Time
	UpdatedAt time.Time
}

const HelloTableName = "hello"

func (h Hello) TableName() string { return HelloTableName }
