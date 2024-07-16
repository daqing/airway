package user_api

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/core/api/action_api"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/sql_orm"

	"github.com/gin-gonic/gin"
)

type ToggleFollowParams struct {
	UserId int64 `form:"id"`
}

func ToggleFollowAction(c *gin.Context) {
	var p ToggleFollowParams

	if err := c.BindQuery(&p); err != nil {
		api_resp.LogError(c, err)
		return
	}

	user, err := sql_orm.FindOne[models.User]([]string{
		"id",
	}, []sql_orm.KVPair{
		sql_orm.KV("id", p.UserId),
	})

	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	if user == nil {
		api_resp.LogError(c, fmt.Errorf("the followed user must exists"))
		return
	}

	currentUser := CurrentUser(c.GetHeader("X-Auth-Token"))
	if currentUser == nil {
		api_resp.LogInvalidUser(c)
		return
	}

	count, err := action_api.ToggleAction(currentUser.ID, models.ActionFavorite, user)

	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	api_resp.OK(c, gin.H{"count": count, "id": p.UserId})
}
