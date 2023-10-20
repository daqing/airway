package comment_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/plugins/user_plugin"
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
		utils.LogError(c, err)
		return
	}

	currentUser := user_plugin.UserFromAuthToken(c.GetHeader("X-Auth-Token"))
	if currentUser == nil {
		utils.LogInvalidUserId(c)
		return
	}

	comment, err := CreateComment(currentUser, p.TargetType, p.TargetId, p.Content)
	if err != nil {
		utils.LogError(c, err)
		return
	}

	item := repo.ItemResp[Comment, CommentResp](comment)

	resp.OK(c, gin.H{"comment": item})
}
