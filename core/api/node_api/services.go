package node_api

import (
	"github.com/daqing/airway/lib/repo"
)

func CreateNode(name, key, parentKey string, level int) (*Node, error) {
	return repo.Insert[Node](
		[]repo.KVPair{
			repo.KV("name", name),
			repo.KV("key", key),
			repo.KV("parent_key", parentKey),
			repo.KV("level", level),
		},
	)
}

func Nodes(fields []string, order string, page, limit int) ([]*Node, error) {
	if page == 0 {
		page = 1
	}

	return repo.FindLimit[Node](
		fields,
		[]repo.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}
