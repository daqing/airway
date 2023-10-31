package post_api

import (
	"fmt"

	"github.com/daqing/airway/api/action_api"
	"github.com/daqing/airway/api/tag_api"
	"github.com/daqing/airway/lib/repo"
)

func CreatePost(title, content string, user_id, node_id int64, fee int, tags []string) (*Post, error) {
	if len(title) == 0 {
		return nil, fmt.Errorf("title can't be empty")
	}

	if len(content) == 0 {
		return nil, fmt.Errorf("content can't be empty")
	}

	if user_id <= 0 {
		return nil, fmt.Errorf("user_id must be greater than zero")
	}

	if node_id <= 0 {
		return nil, fmt.Errorf("node_id must be greater than zero")
	}

	post, err := repo.Insert[Post](
		[]repo.KVPair{
			repo.KV("user_id", user_id),
			repo.KV("node_id", node_id),
			repo.KV("title", title),
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

func TogglePostAction(userId int64, action string, postId int64) (int64, error) {
	post, err := repo.FindRow[Post]([]string{"id"}, []repo.KVPair{
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