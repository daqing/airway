package user_api

const tableName = "users"

func (u User) TableName() string { return tableName }
