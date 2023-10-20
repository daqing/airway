package post_plugin

import (
	"github.com/daqing/airway/lib/utils"
)

type PostResp struct {
	Id        int64
	UserId    int64
	NodeId    int64
	Title     string
	Content   string
	Fee       int
	CreatedAt utils.Timestamp
	UpdatedAt utils.Timestamp
}

func (pr PostResp) Fields() []string {
	return []string{"id", "user_id", "node_id", "title", "content", "fee"}
}