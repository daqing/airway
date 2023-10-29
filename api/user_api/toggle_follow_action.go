package user_api

import (
	"fmt"

	"github.com/daqing/airway/api/action_api"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"

	"github.com/gin-gonic/gin"
)

type ToggleFollowParams struct {
	UserId int64 `form:"id"`
}

func ToggleFollowAction(c *gin.Context) {
	var p ToggleFollowParams

	if err := c.BindQuery(&p); err != nil {
		resp.LogError(c, err)
		return
	}

	user, err := repo.FindRow[User]([]string{
		"id",
	}, []repo.KVPair{
		repo.KV("id", p.UserId),
	})

	if err != nil {
		resp.LogError(c, err)
		return
	}

	if user == nil {
		resp.LogError(c, fmt.Errorf("the followed user must exists"))
		return
	}

	currentUser := CurrentUser(c.GetHeader("X-Auth-Token"))
	if currentUser == nil {
		resp.LogInvalidUser(c)
		return
	}

	count, err := action_api.ToggleAction(currentUser.Id, action_api.ActionFavorite, user)

	if err != nil {
		resp.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"count": count, "id": p.UserId})
}
