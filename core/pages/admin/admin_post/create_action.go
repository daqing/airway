package admin_post

import (
	"fmt"

	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	Title      string `form:"title"`
	CustomPath string `form:"custom_path"`
	Content    string `form:"content"`
	Place      string `form:"place"`
	NodeId     int64  `form:"node_id"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	title := utils.TrimFull(p.Title)
	content := utils.TrimFull(p.Content)
	customPath := utils.TrimFull(p.CustomPath)
	place := utils.TrimFull(p.Place)

	if len(title) == 0 || len(content) == 0 || len(place) == 0 {
		page_resp.Error(c, fmt.Errorf("title or content or place must not be empty"))
		return
	}

	token, err := utils.CookieToken(c)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	admin := user_api.CurrentAdmin(token)

	_, err = post_api.CreatePost(title, customPath, place, content, admin.Id, p.NodeId, 0, []string{})
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	page_resp.Redirect(c, "/admin/post")
}
