package node_plugin

import "github.com/daqing/airway/lib/utils"

type NodeResp struct {
	Id        int64
	Name      string
	Key       string
	CreatedAt utils.Timestamp
	UpdatedAt utils.Timestamp
}

func (ur NodeResp) Fields() []string {
	return []string{"id", "name", "key"}
}
