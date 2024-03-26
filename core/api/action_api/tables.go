package action_api

const actionTableName = "actions"

func (a Action) TableName() string { return actionTableName }
