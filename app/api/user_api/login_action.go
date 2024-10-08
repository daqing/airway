package user_api

import (
	"github.com/daqing/airway/app/repos/user_repo"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/orm"
	"github.com/gin-gonic/gin"
)

type LoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func LoginAction(c *gin.Context) {
	var p LoginParams

	if err := c.BindJSON(&p); err != nil {
		api_resp.LogError(c, err)
		return
	}

	user, err := user_repo.LoginUser(
		orm.Eq("username", p.Username),
		p.Password,
	)

	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	api_resp.OK(c, gin.H{"user": user})
}
