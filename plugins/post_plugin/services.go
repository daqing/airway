package post_plugin

import (
	"fmt"

	"github.com/daqing/airway/lib/repo"
)

func CreatePost(title, content string, user_id, node_id int64) (*Post, error) {
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

	return repo.Insert[Post](
		[]repo.KeyValueField{
			repo.NewKV("user_id", user_id),
			repo.NewKV("node_id", node_id),
			repo.NewKV("title", title),
			repo.NewKV("content", content),
		},
	)
}
