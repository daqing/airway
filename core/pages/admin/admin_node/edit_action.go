package admin_node

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

func EditAction(c *gin.Context) {
	id := c.Query("id")

	node, err := repo.FindOne[models.Node](
		[]string{"id", "name", "key"},
		[]repo.KVPair{
			repo.KV("id", id),
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
