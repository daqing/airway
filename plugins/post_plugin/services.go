package post_plugin

import (
	"fmt"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/plugins/action_plugin"
	"github.com/daqing/airway/plugins/tag_plugin"
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
		[]repo.KeyValueField{
			repo.NewKV("user_id", user_id),
			repo.NewKV("node_id", node_id),
			repo.NewKV("title", title),
			repo.NewKV("content", content),
			repo.NewKV("fee", fee),
		},
	)

	if err != nil {
		return nil, err
	}

	for _, tag := range tags {
		err = tag_plugin.CreateTagRelation(tag, post)
		if err != nil {
			return nil, err
		}
	}

	return post, nil
}

func TogglePostAction(postId, userId int64, action string) (int64, error) {
	post, err := repo.FindRow[Post]([]string{"id"}, []repo.KeyValueField{
		repo.NewKV("id", postId),
	})

	if err != nil {
		return repo.InvalidCount, err
	}

	if post == nil {
		// post not found
		return repo.InvalidCount, nil
	}

	return action_plugin.ToggleAction(userId, action, post)

}
