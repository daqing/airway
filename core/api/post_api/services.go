package post_api

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/core/api/action_api"
	"github.com/daqing/airway/core/api/tag_api"
	"github.com/daqing/airway/lib/repo"
)

func Posts(fields []string, place, order string, page, limit int) ([]*models.Post, error) {
	if page == 0 {
		page = 1
	}

	where := []repo.KVPair{}

	if len(place) > 0 {
		where = append(where, repo.KV("place", place))
	}

	return repo.FindLimit[models.Post](
		fields,
		where,
		order,
		(page-1)*limit,
		limit,
	)
}

func CreatePost(title, customPath, place, content string, user_id, node_id uint, fee int, tags []string) (*models.Post, error) {
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

	post, err := repo.Insert[models.Post](
		[]repo.KVPair{
			repo.KV("user_id", user_id),
			repo.KV("node_id", node_id),
			repo.KV("title", title),
			repo.KV("custom_path", customPath),
			repo.KV("place", place),
			repo.KV("content", content),
			repo.KV("fee", fee),
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

func TogglePostAction(userId uint, action string, postId uint) (int64, error) {
	post, err := repo.FindOne[models.Post]([]string{"id"}, []repo.KVPair{
		repo.KV("id", postId),
	})

	if err != nil {
		return repo.InvalidCount, err
	}

	if post == nil {
		// post not found
		return repo.InvalidCount, nil
	}

	return action_api.ToggleAction(userId, action, post)

}
