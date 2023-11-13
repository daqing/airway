package post_page

import (
	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

func NewAction(c *gin.Context) {
	nodes, err := repo.Find[node_api.Node](
		[]string{"id", "name"},
		[]repo.KVPair{
			repo.KV("place", "blog"),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Nodes": nodes,
	}

	page_resp.Page(c, "core", "admin/post", "new", data)
}
