package post_api

import (
	"time"

	"github.com/daqing/airway/core/api/comment_api"
	"github.com/daqing/airway/core/api/node_api"
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

func (p *Post) User() *user_api.User {
	user, err := repo.FindRow[user_api.User](
		[]string{"id", "nickname", "username", "avatar"},
		[]repo.KVPair{
			repo.KV("id", p.UserId),
		},
	)

	if err != nil {
		return nil
	}

	return user
}

func (p *Post) UserAvatar() string {
	user := p.User()

	if user == nil {
		return repo.EMPTY_STRING
	}

	return user.Avatar
}

func (p *Post) Node() *node_api.Node {
	node, err := repo.FindRow[node_api.Node](
		[]string{"id", "name", "key"},
		[]repo.KVPair{
			repo.KV("id", p.NodeId),
		},
	)

	if err != nil {
		return nil
	}

	return node
}

func (p *Post) Comments() ([]*comment_api.Comment, error) {
	return repo.Find[comment_api.Comment](
		[]string{"id", "user_id", "content"},
		[]repo.KVPair{
			repo.KV("target_type", p.PolyType()),
			repo.KV("target_id", p.PolyId()),
		},
	)
}
