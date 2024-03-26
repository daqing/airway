package comment_api

import (
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/pg_repo"
)

func CreateComment(currentUser *user_api.User, polyModel pg_repo.PolyModel, content string) (*Comment, error) {
	return createComment(currentUser, polyModel.PolyType(), polyModel.PolyId(), content)
}

func createComment(currentUser *user_api.User, targetType string, targetId int64, content string) (*Comment, error) {
	return pg_repo.Insert[Comment]([]pg_repo.KVPair{
		pg_repo.KV("user_id", currentUser.Id),
		pg_repo.KV("target_type", targetType),
		pg_repo.KV("target_id", targetId),
		pg_repo.KV("content", content),
	})
}
