package post_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/gin-gonic/gin"
)

type ToggleFavoriteParams struct {
	PostId int64 `form:"id"`
}

func ToggleFavoriteAction(c *gin.Context) {
	var p ToggleLikeParams

	if err := c.ShouldBind(&p); err != nil {
		api_resp.LogError(c, err)
		return
	}

	user := user_api.CurrentUser(c.GetHeader("X-Auth-Token"))
	if user == nil {
		api_resp.LogInvalidUser(c)
		return
	}

	count, err := TogglePostAction(user.ID, models.ActionFavorite, p.PostId)
	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	api_resp.OK(c, gin.H{"id": p.PostId, "count": count})
}
