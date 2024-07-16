package forum

import (
	"time"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func NewAction(c *gin.Context) {
	nodes, err := sql_orm.Find[models.Node](
		[]string{"id", "name", "key"},
		[]sql_orm.KVPair{
			sql_orm.KV("place", "forum"),
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
				ID:   node.ID,
				Name: node.Name,
				URL:  "/forum/node/" + node.Key,
			})
	}

	token, _ := utils.CookieToken(c)
	currentUser := user_api.CurrentUser(token)

	data := map[string]any{
		"Title":    ForumTitle(),
		"Tagline":  ForumTagline(),
		"Year":     time.Now().Year(),
		"RootPath": rootPath,
		"Nodes":    nodeItems,
		"Session":  SessionData(currentUser),
	}

	page_resp.Page(c, "core", "forum!", "new", data)
}
