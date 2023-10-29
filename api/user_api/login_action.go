package user_api

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

type LoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginAction(c *gin.Context) {
	var p LoginParams

	if err := c.BindJSON(&p); err != nil {
		resp.LogError(c, err)
		return
	}

	user, err := LoginUser(p.Username, p.Password)
	if err != nil {
		resp.LogError(c, err)
		return
	}

	item := repo.ItemResp[User, UserResp](user)
	resp.OK(c, gin.H{"user": item})
}
