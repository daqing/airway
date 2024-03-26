package user_api

import (
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/pg_repo"
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

	user, err := LoginUser(
		[]pg_repo.KVPair{pg_repo.KV("username", p.Username)},
		p.Password,
	)

	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	item := pg_repo.ItemResp[User, UserResp](user)
	api_resp.OK(c, gin.H{"user": item})
}
