package setting_api

const tableName = "settings"

func (s Setting) TableName() string { return tableName }
