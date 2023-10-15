package user_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	users, err := repo.Find[User](
		[]string{"id", "username", "nickname", "role"},
		[]repo.KeyValueField{},
	)

	if err != nil {
		resp.Error(c, err)
		return
	}

	resp.OK(c, gin.H{"list": users})
}
