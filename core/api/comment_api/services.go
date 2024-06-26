package comment_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/app/services"
	"github.com/daqing/airway/lib/repo"
)

func CreateComment(currentUser *models.User, polyModel services.PolyModel, content string) (*models.Comment, error) {
	return createComment(currentUser, polyModel.PolyType(), polyModel.PolyId(), content)
}

func createComment(currentUser *models.User, targetType string, targetId uint, content string) (*models.Comment, error) {
	return repo.Insert[models.Comment]([]repo.KVPair{
		repo.KV("user_id", currentUser.ID),
		repo.KV("target_type", targetType),
		repo.KV("target_id", targetId),
		repo.KV("content", content),
	})
}
