package user_api

import (
	"github.com/daqing/airway/lib/api_resp"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	Nickname string `json:"nickname"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.BindJSON(&p); err != nil {
		api_resp.LogError(c, err)
		return
	}

	user, err := CreateBasicUser(p.Nickname, p.Username, p.Password)
	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	api_resp.OK(c, gin.H{"user": user})
}
