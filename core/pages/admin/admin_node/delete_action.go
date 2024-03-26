package admin_node

import (
	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/pg_repo"
	"github.com/gin-gonic/gin"
)

func DeleteAction(c *gin.Context) {
	id := c.Query("id")

	err := pg_repo.Delete[node_api.Node](
		[]pg_repo.KVPair{
			pg_repo.KV("id", id),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	page_resp.Redirect(c, "/admin/node")
}
