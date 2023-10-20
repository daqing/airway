package post_plugin

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/plugins/action_plugin"
	"github.com/daqing/airway/plugins/user_plugin"
	"github.com/gin-gonic/gin"
)

type ToggleFavoriteParams struct {
	PostId int64 `form:"id"`
}

func ToggleFavoriteAction(c *gin.Context) {
	var p ToggleLikeParams

	if err := c.ShouldBind(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	user := user_plugin.CurrentUser(c.GetHeader("X-Auth-Token"))
	if user == nil {
		utils.LogInvalidUser(c)
		return
	}

	count, err := TogglePostAction(user.Id, action_plugin.ActionFavorite, p.PostId)
	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"id": p.PostId, "count": count})
}
