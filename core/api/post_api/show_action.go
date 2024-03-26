package post_api

import (
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/pg_repo"
	"github.com/gin-gonic/gin"
)

type ShowParams struct {
	Id int64 `form:"id"`
}

func ShowAction(c *gin.Context) {
	var p ShowParams

	if err := c.ShouldBind(&p); err != nil {
		api_resp.LogError(c, err)
		return
	}

	post, err := pg_repo.FindRow[Post]([]string{
		"id", "user_id", "node_id", "title", "content",
	}, []pg_repo.KVPair{
		pg_repo.KV("id", p.Id),
	})

	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	api_resp.OK(c, gin.H{"post": post})
}
