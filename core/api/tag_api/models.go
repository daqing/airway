package tag_api

import "time"

type Tag struct {
	Id int64

	Name string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type TagRelation struct {
	TagId        int64
	RelationType string
	RelationId   int64
}
