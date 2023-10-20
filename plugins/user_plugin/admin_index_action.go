package user_plugin

import (
	"fmt"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func AdminIndexAction(c *gin.Context) {
	admin := CurrentAdmin(c.GetHeader("X-Auth-Token"))
	if admin == nil {
		utils.LogError(c, fmt.Errorf("current admin not found"))
		return
	}

	list, err := repo.ListResp[User, UserResp]()
	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"list": list})
}
