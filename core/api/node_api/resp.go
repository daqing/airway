package node_api

import "github.com/daqing/airway/lib/repo"

type NodeResp struct {
	Id int64

	Name      string
	Key       string
	ParentKey string
	Level     int

	CreatedAt repo.Timestamp
	UpdatedAt repo.Timestamp
}

func (ur NodeResp) Fields() []string {
	return []string{"id", "name", "key", "parent_id", "level"}
}
