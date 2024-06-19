package models

import (
	"github.com/daqing/airway/lib/repo"
	"gorm.io/gorm"
)

type Comment struct {
	gorm.Model

	UserId uint

	TargetId   uint
	TargetType string
	Content    string
}

func (c *Comment) User() *User {
	user, err := repo.FindOne[User](
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

func (c Comment) TableName() string { return "comments" }
