package node_api

import "time"

type Node struct {
	Id int64

	ParentId int64

	Name string
	Key  string

	Level int
	Place string

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "nodes"

func (n Node) TableName() string { return tableName }
