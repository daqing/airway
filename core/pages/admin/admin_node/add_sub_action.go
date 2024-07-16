package admin_node

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/gin-gonic/gin"
)

func AddSubAction(c *gin.Context) {
	id := c.Query("id")

	node, err := sql_orm.FindOne[models.Node](
		[]string{"id", "name"},
		[]sql_orm.KVPair{
			sql_orm.KV("id", id),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Node": node,
	}

	page_resp.Page(c, "core", "admin.node", "add_sub", data)
}
