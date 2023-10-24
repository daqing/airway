package post_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type ShowParams struct {
	Id int64 `form:"id"`
}

func ShowAction(c *gin.Context) {
	var p ShowParams

	if err := c.ShouldBind(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	post, err := repo.FindRow[Post]([]string{
		"id", "user_id", "node_id", "title", "content",
	}, []repo.KVPair{
		repo.KV("id", p.Id),
	})

	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"post": post})
}
