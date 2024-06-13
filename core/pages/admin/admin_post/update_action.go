package admin_post

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

type UpdateParams struct {
	Id         uint   `form:"id"`
	Title      string `form:"title"`
	Content    string `form:"content"`
	Place      string `form:"place"`
	NodeId     string `form:"node_id"`
	CustomPath string `form:"custom_path"`
}

func UpdateAction(c *gin.Context) {
	var p UpdateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	ok := repo.UpdateFields[models.Post](
		p.Id,
		[]repo.KVPair{
			repo.KV("title", p.Title),
			repo.KV("content", p.Content),
			repo.KV("place", p.Place),
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
