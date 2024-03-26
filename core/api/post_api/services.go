package post_api

import (
	"fmt"

	"github.com/daqing/airway/core/api/action_api"
	"github.com/daqing/airway/core/api/tag_api"
	"github.com/daqing/airway/lib/pg_repo"
)

func Posts(fields []string, place, order string, page, limit int) ([]*Post, error) {
	if page == 0 {
		page = 1
	}

	where := []pg_repo.KVPair{}

	if len(place) > 0 {
		where = append(where, pg_repo.KV("place", place))
	}

	return pg_repo.FindLimit[Post](
		fields,
		where,
		order,
		(page-1)*limit,
		limit,
	)
}

func CreatePost(title, customPath, place, content string, user_id, node_id int64, fee int, tags []string) (*Post, error) {
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

	post, err := pg_repo.Insert[Post](
		[]pg_repo.KVPair{
			pg_repo.KV("user_id", user_id),
			pg_repo.KV("node_id", node_id),
			pg_repo.KV("title", title),
			pg_repo.KV("custom_path", customPath),
			pg_repo.KV("place", place),
			pg_repo.KV("content", content),
			pg_repo.KV("fee", fee),
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

func TogglePostAction(userId int64, action string, postId int64) (int64, error) {
	post, err := pg_repo.FindRow[Post]([]string{"id"}, []pg_repo.KVPair{
		pg_repo.KV("id", postId),
	})

	if err != nil {
		return pg_repo.InvalidCount, err
	}

	if post == nil {
		// post not found
		return pg_repo.InvalidCount, nil
	}

	return action_api.ToggleAction(userId, action, post)

}
