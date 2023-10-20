package post_plugin

import (
	"strings"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/plugins/user_plugin"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	NodeId  int64  `json:"node_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Fee     int    `json:"fee"`
	Tags    string `json:"tags"` // 使用英文逗号分隔
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.BindJSON(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	user := user_plugin.CurrentUser(c.GetHeader("X-Auth-Token"))

	if user == nil {
		utils.LogInvalidUserId(c)
		return
	}

	tags := strings.Split(p.Tags, ",")
	post, err := CreatePost(p.Title, p.Content, user.Id, p.NodeId, p.Fee, tags)
	if err != nil {
		utils.LogError(c, err)
		return
	}

	item := repo.ItemResp[Post, PostResp](post)

	resp.OK(c, gin.H{"post": item})
}
