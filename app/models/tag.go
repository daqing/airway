package models

type Tag struct {
	BaseModel

	Name string
}

func (t Tag) TableName() string { return "tags" }

type TagRelation struct {
	TagId        int64
	RelationType string
	RelationId   int64
}

func (tr TagRelation) TableName() string { return "tag_relations" }
