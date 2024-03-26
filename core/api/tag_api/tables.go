package tag_api

const tableName = "tags"

func (t Tag) TableName() string { return tableName }

const relationTableName = "tag_relation"

func (tr TagRelation) TableName() string { return relationTableName }
