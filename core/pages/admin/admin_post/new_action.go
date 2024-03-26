package admin_post

import (
	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/pg_repo"
	"github.com/gin-gonic/gin"
)

func NewAction(c *gin.Context) {
	nodes, err := pg_repo.Find[node_api.Node](
		[]string{"id", "name"},
		[]pg_repo.KVPair{},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Nodes": nodes,
	}

	page_resp.Page(c, "core", "admin.post", "new", data)
}
