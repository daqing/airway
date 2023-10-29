package post_plugin

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/plugins/action_plugin"
	"github.com/daqing/airway/plugins/user_plugin"
	"github.com/gin-gonic/gin"
)

type ToggleLikeParams struct {
	PostId int64 `form:"id"`
}

func ToggleLikeAction(c *gin.Context) {
	var p ToggleLikeParams

	if err := c.ShouldBind(&p); err != nil {
		resp.LogError(c, err)
		return
	}

	user := user_plugin.CurrentUser(c.GetHeader("X-Auth-Token"))
	if user == nil {
		resp.LogInvalidUser(c)
		return
	}

	count, err := TogglePostAction(user.Id, action_plugin.ActionLike, p.PostId)
	if err != nil {
		resp.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"id": p.PostId, "count": count})
}
