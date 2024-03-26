package node_api

import "github.com/daqing/airway/lib/pg_repo"

type NodeResp struct {
	Id int64

	Name      string
	Key       string
	ParentKey string
	Level     int

	CreatedAt pg_repo.Timestamp
	UpdatedAt pg_repo.Timestamp
}

func (ur NodeResp) Fields() []string {
	return []string{"id", "name", "key", "parent_id", "level"}
}
