package post_api

import (
	"log"
	"time"

	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/repo"
)

type Post struct {
	Id         int64
	UserId     int64
	NodeId     int64
	Title      string
	CustomPath string
	Place      string
	Content    string
	Fee        int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const tableName = "posts"

func (p Post) TableName() string { return tableName }

const polyType = "post"

func (p *Post) PolyId() int64    { return p.Id }
func (p *Post) PolyType() string { return polyType }

func (p *Post) UserAvatar() string {
	user, err := repo.FindRow[user_api.User](
		[]string{"avatar"},
		[]repo.KVPair{
			repo.KV("id", p.UserId),
		},
	)

	if err != nil {
		// TODO: fix logging
		log.Println(err)
		return repo.EMPTY_STRING
	}

	return user.Avatar
}
