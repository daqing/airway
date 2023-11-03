package user_api

import (
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

type LoginAdminParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginAdminAction(c *gin.Context) {
	var p LoginParams

	if err := c.BindJSON(&p); err != nil {
		api_resp.LogError(c, err)
		return
	}

	user, err := LoginAdmin(p.Username, p.Password)
	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	item := repo.ItemResp[User, UserResp](user)
	api_resp.OK(c, gin.H{"user": item})
}
