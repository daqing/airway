package node_api

const tableName = "nodes"

func (n Node) TableName() string { return tableName }
