package node_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
)

func CreateNode(name, key, place string, parentId models.IdType, level int) (*models.Node, error) {
	return sql_orm.Insert[models.Node](
		[]sql_orm.KVPair{
			sql_orm.KV("name", name),
			sql_orm.KV("key", key),
			sql_orm.KV("place", place),
			sql_orm.KV("parent_id", parentId),
			sql_orm.KV("level", level),
		},
	)
}

func Nodes(fields []string, order string, page, limit int) ([]*models.Node, error) {
	if page == 0 {
		page = 1
	}

	return sql_orm.FindLimit[models.Node](
		fields,
		[]sql_orm.KVPair{},
		order,
		(page-1)*limit,
		limit,
	)
}

func NameMap(place string) (map[models.IdType]string, error) {
	nodes, err := sql_orm.Find[models.Node](
		[]string{"id", "name"},
		[]sql_orm.KVPair{
			sql_orm.KV("place", place),
		},
	)

	if err != nil {
		return nil, err
	}

	var result = make(map[models.IdType]string)

	for _, node := range nodes {
		result[node.ID] = node.Name
	}

	return result, nil

}
