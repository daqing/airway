package post_page

import (
	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

func EditAction(c *gin.Context) {
	id := c.Query("id")

	post, err := repo.FindRow[post_api.Post](
		[]string{"id", "title", "custom_path", "cat", "content"},
		[]repo.KVPair{
			repo.KV("id", id),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	nodes, err := node_api.Nodes("id DESC", 1, 100)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Post":  post,
		"Nodes": nodes,
	}

	page_resp.Page(c, "core", "admin/post", "edit", data)
}
