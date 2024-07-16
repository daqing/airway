package admin_post

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/gin-gonic/gin"
)

func EditAction(c *gin.Context) {
	id := c.Query("id")

	post, err := sql_orm.FindOne[models.Post](
		[]string{"id", "title", "custom_path", "place", "content"},
		[]sql_orm.KVPair{
			sql_orm.KV("id", id),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	nodes, err := sql_orm.Find[models.Node](
		[]string{"id", "name"},
		[]sql_orm.KVPair{},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Post":  post,
		"Nodes": nodes,
	}

	page_resp.Page(c, "core", "admin.post", "edit", data)
}
