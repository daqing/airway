package node_api

import "github.com/daqing/airway/lib/pg_repo"

func CreateNode(name, key, place string, parentId int64, level int) (*Node, error) {
	return pg_repo.Insert[Node](
		[]pg_repo.KVPair{
			pg_repo.KV("name", name),
			pg_repo.KV("key", key),
			pg_repo.KV("place", place),
			pg_repo.KV("parent_id", parentId),
			pg_repo.KV("level", level),
		},
	)
}

func Nodes(fields []string, order string, page, limit int) ([]*Node, error) {
	if page == 0 {
		page = 1
	}

	return pg_repo.FindLimit[Node](
		fields,
		[]pg_repo.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}

func NameMap(place string) (map[int64]string, error) {
	nodes, err := pg_repo.Find[Node](
		[]string{"id", "name"},
		[]pg_repo.KVPair{
			pg_repo.KV("place", place),
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
