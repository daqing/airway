package checkin_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func CreateAction(c *gin.Context) {
	currentUser := user_api.CurrentUser(c.GetHeader("X-Auth-Token"))
	if currentUser == nil {
		api_resp.LogInvalidUser(c)
		return
	}

	checkin, err := CreateCheckin(currentUser, utils.Today())

	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	item := repo.ItemResp[models.Checkin, CheckinResp](checkin)

	api_resp.OK(c, gin.H{"checkin": item})
}
