package comment_api

import "github.com/daqing/airway/app/models"

type CommentResp struct {
	Id int64

	TargetId   int64
	TargetType string
	Content    string

	CreatedAt models.Timestamp
	UpdatedAt models.Timestamp
}

func (ur CommentResp) Fields() []string {
	return []string{"id", "target_id", "target_type", "content"}
}
