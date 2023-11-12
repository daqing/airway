package tag_api

import "time"

type Tag struct {
	Id int64

	Name string

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "tags"

func (t Tag) TableName() string { return tableName }

type TagRelation struct {
	TagId        int64
	RelationType string
	RelationId   int64
}

const relationTableName = "tag_relation"

func (tr TagRelation) TableName() string { return relationTableName }
