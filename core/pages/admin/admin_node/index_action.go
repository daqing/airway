package admin_node

import (
	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/gin-gonic/gin"
)

type IndexParams struct {
	Page int `form:"page"`
}

func IndexAction(c *gin.Context) {
	var p IndexParams

	if err := c.Bind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	nodes, err := node_api.Nodes(
		[]string{"id", "name", "key", "parent_id", "place", "level"},
		"id DESC",
		p.Page,
		50,
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Nodes": nodes,
	}

	page_resp.Page(c, "core", "admin.node", "index", data)
}
