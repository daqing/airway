package user_plugin

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/plugins/action_plugin"

	"github.com/gin-gonic/gin"
)

type ToggleFollowParams struct {
	UserId int64 `form:"id"`
}

func ToggleFollowAction(c *gin.Context) {
	var p ToggleFollowParams

	if err := c.BindQuery(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	user := UserFromAuthToken(c.GetHeader("X-Auth-Token"))
	if user == nil {
		utils.LogInvalidUserId(c)
		return
	}

	count, err := action_plugin.ToggleAction(user, user.Id, action_plugin.ActionFavorite)

	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"count": count, "id": p.UserId})
}
