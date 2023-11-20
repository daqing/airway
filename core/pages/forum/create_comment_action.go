package forum

import (
	"fmt"

	"github.com/daqing/airway/core/api/comment_api"
	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type CreateCommentParams struct {
	TargetId int64  `form:"target_id"`
	Content  string `form:"content"`
}

func CreateCommentAction(c *gin.Context) {
	var p CreateCommentParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	token, _ := utils.CookieToken(c)
	currentUser := user_api.CurrentUser(token)
	if currentUser == nil {
		page_resp.Redirect(c, "/forum")
		return
	}

	polyModel := &post_api.Post{Id: p.TargetId}

	_, err := comment_api.CreateComment(currentUser, polyModel, p.Content)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	backPath := fmt.Sprintf("/forum/post/%d", p.TargetId)
	page_resp.Redirect(c, backPath)
}
