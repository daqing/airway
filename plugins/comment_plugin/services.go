package comment_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/plugins/user_plugin"
)

func CreateComment(currentUser *user_plugin.User, targetType string, targetId int64, content string) (*Comment, error) {
	return repo.Insert[Comment]([]repo.KeyValueField{
		repo.NewKV("user_id", currentUser.Id),
		repo.NewKV("target_type", targetType),
		repo.NewKV("target_id", targetId),
		repo.NewKV("content", content),
	})
}
