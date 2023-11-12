package post_page

import (
	"fmt"

	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

type UpdateParams struct {
	Id         int64  `form:"id"`
	Title      string `form:"title"`
	Content    string `form:"content"`
	Cat        string `form:"cat"`
	NodeId     string `form:"node_id"`
	CustomPath string `form:"custom_path"`
}

func UpdateAction(c *gin.Context) {
	var p UpdateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	ok := repo.UpdateFields[post_api.Post](
		p.Id,
		[]repo.KVPair{
			repo.KV("title", p.Title),
			repo.KV("content", p.Content),
			repo.KV("cat", p.Cat),
			repo.KV("node_id", p.NodeId),
			repo.KV("custom_path", p.CustomPath),
		},
	)

	if !ok {
		page_resp.Error(c, fmt.Errorf("error updating post with id: %d", p.Id))
		return
	}

	page_resp.Redirect(c, "/admin/post")
}
