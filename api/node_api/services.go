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
