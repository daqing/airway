package tag_plugin

import (
	"fmt"

	"github.com/daqing/airway/lib/repo"
)

func CreateTag(name string) (*Tag, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("name cannot be empty")
	}

	return repo.Insert[Tag]([]repo.KVPair{
		repo.KV("name", name),
	})
}

func CreateTagRelation(tagName string, relation repo.PolyModel) error {
	tags, err := repo.Find[Tag]([]string{"id", "name"}, []repo.KVPair{
		repo.KV("name", tagName),
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

	relations, err := repo.Find[TagRelation]([]string{"id"}, []repo.KVPair{
		repo.KV("tag_id", tag.Id),
		repo.KV("relation_id", relation.PolyId()),
		repo.KV("relation_type", relation.PolyType()),
	})

	if err != nil {
		return err
	}

	if len(relations) == 0 {
		// create new relation
		_, err = repo.Insert[TagRelation]([]repo.KVPair{
			repo.KV("tag_id", tag.Id),
			repo.KV("relation_id", relation.PolyId()),
			repo.KV("relation_type", relation.PolyType()),
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
