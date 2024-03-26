package admin_post

import (
	"fmt"

	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/pg_repo"
	"github.com/gin-gonic/gin"
)

type UpdateParams struct {
	Id         int64  `form:"id"`
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

	ok := pg_repo.UpdateFields[post_api.Post](
		p.Id,
		[]pg_repo.KVPair{
			pg_repo.KV("title", p.Title),
			pg_repo.KV("content", p.Content),
			pg_repo.KV("place", p.Place),
			pg_repo.KV("node_id", p.NodeId),
			pg_repo.KV("custom_path", p.CustomPath),
		},
	)

	if !ok {
		page_resp.Error(c, fmt.Errorf("error updating post with id: %d", p.Id))
		return
	}

	page_resp.Redirect(c, "/admin/post")
}
