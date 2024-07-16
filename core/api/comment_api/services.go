package comment_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/sql_orm"
)

func CreateComment(currentUser *models.User, polyModel models.PolyModel, content string) (*models.Comment, error) {
	return createComment(currentUser, polyModel.PolyType(), polyModel.PolyId(), content)
}

func createComment(currentUser *models.User, targetType string, targetId models.IdType, content string) (*models.Comment, error) {
	return sql_orm.Insert[models.Comment]([]sql_orm.KVPair{
		sql_orm.KV("user_id", currentUser.ID),
		sql_orm.KV("target_type", targetType),
		sql_orm.KV("target_id", targetId),
		sql_orm.KV("content", content),
	})
}
