package models

import (
	"gorm.io/gorm"
)

type Node struct {
	gorm.Model

	ParentId int64

	Name string
	Key  string

	Level int
	Place string
}

func (n Node) TableName() string { return "nodes" }
