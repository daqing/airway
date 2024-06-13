package tag_api

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/repo"
)

func CreateTag(name string) (*models.Tag, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("name cannot be empty")
	}

	return repo.Insert[models.Tag]([]repo.KVPair{
		repo.KV("name", name),
	})
}

func CreateTagRelation(tagName string, relation models.PolyModel) error {
	tags, err := repo.Find[models.Tag]([]string{"id", "name"}, []repo.KVPair{
		repo.KV("name", tagName),
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

	relations, err := repo.Find[models.TagRelation]([]string{"id"}, []repo.KVPair{
		repo.KV("tag_id", tag.ID),
		repo.KV("relation_id", relation.PolyId()),
		repo.KV("relation_type", relation.PolyType()),
	})

	if err != nil {
		return err
	}

	if len(relations) == 0 {
		// create new relation
		_, err = repo.Insert[models.TagRelation]([]repo.KVPair{
			repo.KV("tag_id", tag.ID),
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
