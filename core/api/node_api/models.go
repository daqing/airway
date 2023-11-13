package node_api

import "time"

type Node struct {
	Id int64

	Name      string
	Key       string
	ParentKey string
	Level     int

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "nodes"

func (n Node) TableName() string { return tableName }
