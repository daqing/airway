package models

import "gorm.io/gorm"

type Tag struct {
	gorm.Model

	Name string
}

func (t Tag) TableName() string { return "tags" }

type TagRelation struct {
	TagId        int64
	RelationType string
	RelationId   int64
}

func (tr TagRelation) TableName() string { return "tag_relations" }
