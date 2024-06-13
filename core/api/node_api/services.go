package node_api

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/models"
)

func CreateNode(name, key, place string, parentId uint, level int) (*models.Node, error) {
	return repo.Insert[models.Node](
		[]repo.KVPair{
			repo.KV("name", name),
			repo.KV("key", key),
			repo.KV("place", place),
			repo.KV("parent_id", parentId),
			repo.KV("level", level),
		},
	)
}

func Nodes(fields []string, order string, page, limit int) ([]*models.Node, error) {
	if page == 0 {
		page = 1
	}

	return repo.FindLimit[models.Node](
		fields,
		[]repo.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}

func NameMap(place string) (map[uint]string, error) {
	nodes, err := repo.Find[models.Node](
		[]string{"id", "name"},
		[]repo.KVPair{
			repo.KV("place", place),
		},
	)

	if err != nil {
		return nil, err
	}

	var result = make(map[uint]string)

	for _, node := range nodes {
		result[node.ID] = node.Name
	}

	return result, nil

}
