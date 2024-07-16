package tag_api

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
)

func CreateTag(name string) (*models.Tag, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("name cannot be empty")
	}

	return sql_orm.Insert[models.Tag]([]sql_orm.KVPair{
		sql_orm.KV("name", name),
	})
}

func CreateTagRelation(tagName string, relation models.PolyModel) error {
	tags, err := sql_orm.Find[models.Tag]([]string{"id", "name"}, []sql_orm.KVPair{
		sql_orm.KV("name", tagName),
	})

	if err != nil {
		return err
	}

	var tag *models.Tag

	if len(tags) == 0 {
		// tag not found, create one
		tag, err = CreateTag(tagName)

		if err != nil {
			return err
		}
	} else if len(tags) == 1 {
		tag = tags[0]
	} else {
		return fmt.Errorf("number of tags is wrong: %d", len(tags))
	}

	relations, err := sql_orm.Find[models.TagRelation]([]string{"id"}, []sql_orm.KVPair{
		sql_orm.KV("tag_id", tag.ID),
		sql_orm.KV("relation_id", relation.PolyId()),
		sql_orm.KV("relation_type", relation.PolyType()),
	})

	if err != nil {
		return err
	}

	if len(relations) == 0 {
		// create new relation
		_, err = sql_orm.Insert[models.TagRelation]([]sql_orm.KVPair{
			sql_orm.KV("tag_id", tag.ID),
			sql_orm.KV("relation_id", relation.PolyId()),
			sql_orm.KV("relation_type", relation.PolyType()),
		})

		if err != nil {
			return err
		}
	} else if len(relations) > 1 {
		// wrong number of relations
		return fmt.Errorf("wrong number of relations: %d", len(relations))
	}

	return nil
}
