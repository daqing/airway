package setting_api

import (
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func AdminIndexAction(c *gin.Context) {
	if !user_api.CheckAdmin(c.GetHeader("X-Auth-Token")) {
		resp.LogInvalidAdmin(c)
		return
	}

	list, err := repo.ListResp[Setting, SettingResp]()
	if err != nil {
		resp.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"list": list})
}
