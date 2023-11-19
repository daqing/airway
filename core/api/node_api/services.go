package node_api

import (
	"github.com/daqing/airway/lib/repo"
)

func CreateNode(name, key, place string, parentId int64, level int) (*Node, error) {
	return repo.Insert[Node](
		[]repo.KVPair{
			repo.KV("name", name),
			repo.KV("key", key),
			repo.KV("place", place),
			repo.KV("parent_id", parentId),
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

func NameMap(place string) (map[int64]string, error) {
	nodes, err := repo.Find[Node](
		[]string{"id", "name"},
		[]repo.KVPair{
			repo.KV("place", place),
		},
	)

	if err != nil {
		return nil, err
	}

	var result = make(map[int64]string)

	for _, node := range nodes {
		result[node.Id] = node.Name
	}

	return result, nil

}
