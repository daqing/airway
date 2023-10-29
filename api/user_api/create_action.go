package user_api

import (
	"github.com/daqing/airway/lib/resp"
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
		resp.LogError(c, err)
		return
	}

	user, err := CreateUser(p.Nickname, p.Username, BasicRole, p.Password)
	if err != nil {
		resp.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"user": user})
}
