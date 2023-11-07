package post_api

import (
	"github.com/daqing/airway/lib/utils"
)

type PostResp struct {
	Id         int64
	UserId     int64
	NodeId     int64
	Title      string
	CustomPath string
	Cat        string
	Content    string
	Fee        int
	CreatedAt  utils.Timestamp
	UpdatedAt  utils.Timestamp
}

func (pr PostResp) Fields() []string {
	return []string{
		"id", "user_id", "node_id",
		"title", "custom_path",
		"cat", "content", "fee",
	}
}
