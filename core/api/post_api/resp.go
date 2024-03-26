package post_api

import "github.com/daqing/airway/lib/pg_repo"

type PostResp struct {
	Id         int64
	UserId     int64
	NodeId     int64
	Title      string
	CustomPath string
	Place      string
	Content    string
	Fee        int
	CreatedAt  pg_repo.Timestamp
	UpdatedAt  pg_repo.Timestamp
}

func (pr PostResp) Fields() []string {
	return []string{
		"id", "user_id", "node_id",
		"title", "custom_path",
		"place", "content", "fee",
	}
}
