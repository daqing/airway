package admin_post

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

func NewAction(c *gin.Context) {
	nodes, err := repo.Find[models.Node](
		[]string{"id", "name"},
		[]repo.KVPair{},
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
