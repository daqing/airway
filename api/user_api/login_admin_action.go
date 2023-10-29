package user_api

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

type LoginAdminParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginAdminAction(c *gin.Context) {
	var p LoginParams

	if err := c.BindJSON(&p); err != nil {
		resp.LogError(c, err)
		return
	}

	user, err := LoginAdmin(p.Username, p.Password)
	if err != nil {
		resp.LogError(c, err)
		return
	}

	item := repo.ItemResp[User, UserResp](user)
	resp.OK(c, gin.H{"user": item})
}
