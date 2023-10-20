package user_plugin

import (
	"fmt"

	"github.com/daqing/airway/lib/repo"
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

	user, err := repo.FindRow[User]([]string{
		"id",
	}, []repo.KeyValueField{
		repo.NewKV("id", p.UserId),
	})

	if err != nil {
		utils.LogError(c, err)
		return
	}

	if user == nil {
		utils.LogError(c, fmt.Errorf("the followed user must exists"))
		return
	}

	currentUser := CurrentUser(c.GetHeader("X-Auth-Token"))
	if currentUser == nil {
		utils.LogInvalidUser(c)
		return
	}

	count, err := action_plugin.ToggleAction(currentUser.Id, action_plugin.ActionFavorite, user)

	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"count": count, "id": p.UserId})
}
