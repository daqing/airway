package forum

import (
	"time"

	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func NewAction(c *gin.Context) {
	nodes, err := repo.Find[node_api.Node](
		[]string{"id", "name", "key"},
		[]repo.KVPair{
			repo.KV("place", "forum"),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	rootPath := utils.PathPrefix("forum")

	nodeItems := []*NodeItem{}

	for _, node := range nodes {
		nodeItems = append(nodeItems,
			&NodeItem{
				Id:   node.Id,
				Name: node.Name,
				URL:  "/forum/node/" + node.Key,
			})
	}

	token, _ := c.Cookie("user_api_token")

	data := map[string]any{
		"Title":    ForumTitle(),
		"Tagline":  ForumTagline(),
		"Year":     time.Now().Year(),
		"RootPath": rootPath,
		"Nodes":    nodeItems,
		"Session":  SessionData(token),
	}

	page_resp.Page(c, "core", "forum!", "new", data)
}
