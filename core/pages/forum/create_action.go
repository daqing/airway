package forum

import (
	"fmt"

	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/models"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	Title   string `form:"title"`
	Content string `form:"content"`
	NodeId  uint   `form:"node_id"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	title := utils.TrimFull(p.Title)
	content := utils.TrimFull(p.Content)

	if len(title) == 0 || len(content) == 0 {
		page_resp.Error(c, fmt.Errorf("title and content must be specified"))
		return
	}

	ex, err := repo.Exists[models.Node]([]repo.KVPair{repo.KV("id", p.NodeId)})
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	if !ex {
		page_resp.Error(c, fmt.Errorf("node not exists for id: %d", p.NodeId))
		return
	}

	token, _ := utils.CookieToken(c)

	currentUser := user_api.CurrentUser(token)
	if currentUser == nil {
		page_resp.Error(c, fmt.Errorf("user is not logged in"))
		return
	}

	_, err = post_api.CreatePost(
		title, "", "forum",
		content,
		currentUser.ID, p.NodeId,
		0,
		[]string{},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	rootPath := utils.PathPrefix("forum")

	page_resp.Redirect(c, rootPath)
}
