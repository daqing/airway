package post_api

import (
	"github.com/daqing/airway/core/api/action_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/gin-gonic/gin"
)

type ToggleLikeParams struct {
	PostId int64 `form:"id"`
}

func ToggleLikeAction(c *gin.Context) {
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

	count, err := TogglePostAction(user.Id, action_api.ActionLike, p.PostId)
	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	api_resp.OK(c, gin.H{"id": p.PostId, "count": count})
}
