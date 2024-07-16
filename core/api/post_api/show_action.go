package post_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/gin-gonic/gin"
)

type ShowParams struct {
	Id models.IdType `form:"id"`
}

func ShowAction(c *gin.Context) {
	var p ShowParams

	if err := c.ShouldBind(&p); err != nil {
		api_resp.LogError(c, err)
		return
	}

	post, err := sql_orm.FindOne[models.Post]([]string{
		"id", "user_id", "node_id", "title", "content",
	}, []sql_orm.KVPair{
		sql_orm.KV("id", p.Id),
	})

	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	api_resp.OK(c, gin.H{"post": post})
}
