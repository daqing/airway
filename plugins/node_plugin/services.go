package node_plugin

import "github.com/daqing/airway/lib/repo"

func CreateNode(name, key string) (*Node, error) {
	return repo.Insert[Node](
		[]repo.KeyValueField{
			repo.NewKV("name", name),
			repo.NewKV("key", key),
		},
	)
}
