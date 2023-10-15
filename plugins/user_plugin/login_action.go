package user_plugin

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type LoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginAction(c *gin.Context) {
	var p LoginParams

	if err := c.BindJSON(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	user, err := LoginUser(p.Username, p.Password)
	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"user": user})
}
