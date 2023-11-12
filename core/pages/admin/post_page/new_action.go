package post_page

import (
	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/gin-gonic/gin"
)

func NewAction(c *gin.Context) {
	nodes, err := node_api.Nodes("id DESC", 1, 100)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Nodes": nodes,
	}

	page_resp.Page(c, "core", "admin/post", "new", data)
}
