package comment_api

const tableName = "comments"

func (c Comment) TableName() string { return tableName }
