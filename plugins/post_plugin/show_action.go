package post_plugin

import (
	"fmt"

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

	posts, err := repo.Find[Post]([]string{
		"id", "user_id", "node_id", "title", "content",
	}, []repo.KeyValueField{
		repo.NewKV("id", p.Id),
	})

	if err != nil {
		utils.LogError(c, err)
		return
	}

	if len(posts) != 1 {
		utils.LogError(c, fmt.Errorf("posts should be one record"))
		return
	}

	resp.OK(c, posts[0])
}
