package post_page

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
	Cat        string `form:"cat"`
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
	cat := utils.TrimFull(p.Cat)

	if len(title) == 0 || len(content) == 0 || len(cat) == 0 {
		page_resp.Error(c, fmt.Errorf("title or content or cat must not be empty"))
		return
	}

	token, err := c.Cookie("user_api_token")
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	admin := user_api.CurrentAdmin(token)

	_, err = post_api.CreatePost(title, customPath, cat, content, admin.Id, p.NodeId, 0, []string{})
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	page_resp.Redirect(c, "/admin/post")
}