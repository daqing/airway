package checkin_api

import (
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"

	"github.com/gin-gonic/gin"
)

func CreateAction(c *gin.Context) {
	currentUser := user_api.CurrentUser(c.GetHeader("X-Auth-Token"))
	if currentUser == nil {
		resp.LogInvalidUser(c)
		return
	}

	checkin, err := CreateCheckin(currentUser, utils.Today())

	if err != nil {
		resp.LogError(c, err)
		return
	}

	item := repo.ItemResp[Checkin, CheckinResp](checkin)

	resp.OK(c, gin.H{"checkin": item})
}
