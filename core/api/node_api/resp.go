package node_api

import "github.com/daqing/airway/app/models"

type NodeResp struct {
	Id int64

	Name      string
	Key       string
	ParentKey string
	Level     int

	CreatedAt models.Timestamp
	UpdatedAt models.Timestamp
}

func (ur NodeResp) Fields() []string {
	return []string{"id", "name", "key", "parent_id", "level"}
}
