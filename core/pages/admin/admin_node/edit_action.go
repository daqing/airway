package admin_node

import (
	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/pg_repo"
	"github.com/gin-gonic/gin"
)

func EditAction(c *gin.Context) {
	id := c.Query("id")

	node, err := pg_repo.FindRow[node_api.Node](
		[]string{"id", "name", "key"},
		[]pg_repo.KVPair{
			pg_repo.KV("id", id),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Node": node,
	}

	page_resp.Page(c, "core", "admin.node", "edit", data)
}
