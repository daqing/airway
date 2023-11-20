package comment_api

import (
	"time"

	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/repo"
)

type Comment struct {
	Id int64

	UserId int64

	TargetId   int64
	TargetType string
	Content    string

	CreatedAt time.Time
	UpdatedAt time.Time
}

const tableName = "comments"

func (c Comment) TableName() string { return tableName }

func (c *Comment) User() *user_api.User {
	user, err := repo.FindRow[user_api.User](
		[]string{"id", "nickname", "username", "avatar"},
		[]repo.KVPair{
			repo.KV("id", c.UserId),
		},
	)

	if err != nil {
		return nil
	}

	return user
}
