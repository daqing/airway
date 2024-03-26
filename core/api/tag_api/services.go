package tag_api

import (
	"fmt"

	"github.com/daqing/airway/lib/pg_repo"
)

func CreateTag(name string) (*Tag, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("name cannot be empty")
	}

	return pg_repo.Insert[Tag]([]pg_repo.KVPair{
		pg_repo.KV("name", name),
	})
}

func CreateTagRelation(tagName string, relation pg_repo.PolyModel) error {
	tags, err := pg_repo.Find[Tag]([]string{"id", "name"}, []pg_repo.KVPair{
		pg_repo.KV("name", tagName),
	})

	if err != nil {
		return err
	}

	var tag *Tag

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

	relations, err := pg_repo.Find[TagRelation]([]string{"id"}, []pg_repo.KVPair{
		pg_repo.KV("tag_id", tag.Id),
		pg_repo.KV("relation_id", relation.PolyId()),
		pg_repo.KV("relation_type", relation.PolyType()),
	})

	if err != nil {
		return err
	}

	if len(relations) == 0 {
		// create new relation
		_, err = pg_repo.Insert[TagRelation]([]pg_repo.KVPair{
			pg_repo.KV("tag_id", tag.Id),
			pg_repo.KV("relation_id", relation.PolyId()),
			pg_repo.KV("relation_type", relation.PolyType()),
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
