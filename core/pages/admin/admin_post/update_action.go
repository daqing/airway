package admin_post

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/gin-gonic/gin"
)

type UpdateParams struct {
	Id         models.IdType `form:"id"`
	Title      string        `form:"title"`
	Content    string        `form:"content"`
	Place      string        `form:"place"`
	NodeId     string        `form:"node_id"`
	CustomPath string        `form:"custom_path"`
}

func UpdateAction(c *gin.Context) {
	var p UpdateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	ok := sql_orm.UpdateFields[models.Post](
		p.Id,
		[]sql_orm.KVPair{
			sql_orm.KV("title", p.Title),
			sql_orm.KV("content", p.Content),
			sql_orm.KV("place", p.Place),
			sql_orm.KV("node_id", p.NodeId),
			sql_orm.KV("custom_path", p.CustomPath),
		},
	)

	if !ok {
		page_resp.Error(c, fmt.Errorf("error updating post with id: %d", p.Id))
		return
	}

	page_resp.Redirect(c, "/admin/post")
}
