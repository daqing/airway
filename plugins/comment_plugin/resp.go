package comment_plugin

import "github.com/daqing/airway/lib/utils"

type CommentResp struct {
	Id int64

	TargetId   int64
	TargetType string
	Content    string

	CreatedAt utils.Timestamp
	UpdatedAt utils.Timestamp
}

func (ur CommentResp) Fields() []string {
	return []string{"id", "target_id", "target_type", "content"}
}
