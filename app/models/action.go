package models

type Action struct {
	BaseModel

	UserId     int64
	Action     string
	TargetType string
	TargetId   int64
}

const ActionLike = "like"
const ActionFollow = "follow"
const ActionFavorite = "favorite"

const actionTableName = "actions"

func (a Action) TableName() string { return actionTableName }
