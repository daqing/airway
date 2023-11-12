package setting_api

import (
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/repo"

	"github.com/gin-gonic/gin"
)

func AdminIndexAction(c *gin.Context) {
	if !user_api.CheckAdmin(c.GetHeader("X-Auth-Token")) {
		api_resp.LogInvalidAdmin(c)
		return
	}

	list, err := repo.ListResp[Setting, SettingResp]()
	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	api_resp.OK(c, gin.H{"list": list})
}
