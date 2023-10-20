package checkin_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/plugins/user_plugin"
	"github.com/gin-gonic/gin"
)

func CreateAction(c *gin.Context) {
	currentUser := user_plugin.CurrentUser(c.GetHeader("X-Auth-Token"))
	if currentUser == nil {
		utils.LogInvalidUserId(c)
		return
	}

	checkin, err := CreateCheckin(currentUser, utils.Today())

	if err != nil {
		utils.LogError(c, err)
		return
	}

	item := repo.ItemResp[Checkin, CheckinResp](checkin)

	resp.OK(c, gin.H{"checkin": item})
}
