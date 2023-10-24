package comment_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/plugins/user_plugin"
)

func CreateComment(currentUser *user_plugin.User, targetType string, targetId int64, content string) (*Comment, error) {
	return repo.Insert[Comment]([]repo.KVPair{
		repo.KV("user_id", currentUser.Id),
		repo.KV("target_type", targetType),
		repo.KV("target_id", targetId),
		repo.KV("content", content),
	})
}
