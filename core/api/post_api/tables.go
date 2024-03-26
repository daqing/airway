package post_api

const tableName = "posts"

func (p Post) TableName() string { return tableName }
