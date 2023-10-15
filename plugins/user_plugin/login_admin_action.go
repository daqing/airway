package user_plugin

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type LoginAdminParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginAdminAction(c *gin.Context) {
	var p LoginParams

	if err := c.BindJSON(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	user, err := LoginAdmin(p.Username, p.Password)
	if err != nil {
		utils.LogError(c, err)
		return
	}

	// 保护密码字段
	user.EncryptedPassword = ""

	resp.OK(c, gin.H{"user": user})
}
