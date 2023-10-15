package node_plugin

import "time"

type Node struct {
	Id        int64
	Name      string
	Key       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "nodes"

func (n Node) TableName() string { return tableName }
