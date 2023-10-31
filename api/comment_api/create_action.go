package comment_api

import (
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	TargetId   int64  `json:"target_id"`
	TargetType string `json:"target_type"`
	Content    string `json:"content"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.BindJSON(&p); err != nil {
		resp.LogError(c, err)
		return
	}

	currentUser := user_api.CurrentUser(c.GetHeader("X-Auth-Token"))
	if currentUser == nil {
		resp.LogInvalidUser(c)
		return
	}

	comment, err := CreateComment(currentUser, p.TargetType, p.TargetId, p.Content)
	if err != nil {
		resp.LogError(c, err)
		return
	}

	item := repo.ItemResp[Comment, CommentResp](comment)

	resp.OK(c, gin.H{"comment": item})
}