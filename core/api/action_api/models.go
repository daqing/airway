package action_api

import "time"

type Action struct {
	Id int64

	UserId     int64
	Action     string
	TargetType string
	TargetId   int64

	CreatedAt time.Time
	UpdatedAt time.Time
}

const ActionLike = "like"
const ActionFollow = "follow"
const ActionFavorite = "favorite"

const actionTableName = "actions"

func (a Action) TableName() string { return actionTableName }
