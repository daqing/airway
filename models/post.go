package models

import (
	"github.com/daqing/airway/lib/repo"
	"gorm.io/gorm"
)

type Post struct {
	gorm.Model

	UserId     uint
	NodeId     uint
	Title      string
	CustomPath string
	Place      string
	Content    string
	Fee        int
}

func (p Post) TableName() string { return "posts" }

const postPolyType = "post"

func (p *Post) PolyId() uint     { return p.ID }
func (p *Post) PolyType() string { return postPolyType }

func (p *Post) User() *User {
	user, err := repo.FindRow[User](
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

func (p *Post) Node() *Node {
	node, err := repo.FindRow[Node](
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

func (p *Post) Comments() ([]*Comment, error) {
	return repo.Find[Comment](
		[]string{"id", "user_id", "content"},
		[]repo.KVPair{
			repo.KV("target_type", p.PolyType()),
			repo.KV("target_id", p.PolyId()),
		},
	)
}
