package post_api

import (
	"github.com/daqing/airway/api/action_api"
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

type ToggleFavoriteParams struct {
	PostId int64 `form:"id"`
}

func ToggleFavoriteAction(c *gin.Context) {
	var p ToggleLikeParams

	if err := c.ShouldBind(&p); err != nil {
		resp.LogError(c, err)
		return
	}

	user := user_api.CurrentUser(c.GetHeader("X-Auth-Token"))
	if user == nil {
		resp.LogInvalidUser(c)
		return
	}

	count, err := TogglePostAction(user.Id, action_api.ActionFavorite, p.PostId)
	if err != nil {
		resp.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"id": p.PostId, "count": count})
}
