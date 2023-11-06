package node_api

import "github.com/daqing/airway/lib/repo"

func CreateNode(name, key string) (*Node, error) {
	return repo.Insert[Node](
		[]repo.KVPair{
			repo.KV("name", name),
			repo.KV("key", key),
		},
	)
}

func Nodes(order string, page, limit int) ([]*Node, error) {
	if page == 0 {
		page = 1
	}

	return repo.FindLimit[Node](
		[]string{"id", "name", "key"},
		[]repo.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}
