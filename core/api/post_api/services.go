package post_api

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/core/api/action_api"
	"github.com/daqing/airway/core/api/tag_api"
	"github.com/daqing/airway/lib/sql_orm"
)

func Posts(fields []string, place, order string, page, limit int) ([]*models.Post, error) {
	if page == 0 {
		page = 1
	}

	where := []sql_orm.KVPair{}

	if len(place) > 0 {
		where = append(where, sql_orm.KV("place", place))
	}

	return sql_orm.FindLimit[models.Post](
		fields,
		where,
		order,
		(page-1)*limit,
		limit,
	)
}

func CreatePost(title, customPath, place, content string, user_id, node_id models.IdType, fee int, tags []string) (*models.Post, error) {
	if len(title) == 0 {
		return nil, fmt.Errorf("title can't be empty")
	}

	if len(content) == 0 {
		return nil, fmt.Errorf("content can't be empty")
	}

	if len(place) == 0 {
		return nil, fmt.Errorf("place can't be empty")
	}

	if user_id <= 0 {
		return nil, fmt.Errorf("user_id must be greater than zero")
	}

	if node_id <= 0 {
		return nil, fmt.Errorf("node_id must be greater than zero")
	}

	post, err := sql_orm.Insert[models.Post](
		[]sql_orm.KVPair{
			sql_orm.KV("user_id", user_id),
			sql_orm.KV("node_id", node_id),
			sql_orm.KV("title", title),
			sql_orm.KV("custom_path", customPath),
			sql_orm.KV("place", place),
			sql_orm.KV("content", content),
			sql_orm.KV("fee", fee),
		},
	)

	if err != nil {
		return nil, err
	}

	for _, tag := range tags {
		err = tag_api.CreateTagRelation(tag, post)
		if err != nil {
			return nil, err
		}
	}

	return post, nil
}

func TogglePostAction(userId models.IdType, action string, postId models.IdType) (int64, error) {
	post, err := sql_orm.FindOne[models.Post]([]string{"id"}, []sql_orm.KVPair{
		sql_orm.KV("id", postId),
	})

	if err != nil {
		return sql_orm.InvalidCount, err
	}

	if post == nil {
		// post not found
		return sql_orm.InvalidCount, nil
	}

	return action_api.ToggleAction(userId, action, post)

}
