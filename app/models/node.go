package models

type Node struct {
	BaseModel

	ParentId IdType

	Name string
	Key  string

	Level int
	Place string
}

func (n Node) TableName() string { return "nodes" }
