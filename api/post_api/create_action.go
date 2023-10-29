package post_api

import (
	"strings"

	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
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
		resp.LogError(c, err)
		return
	}

	user := user_api.CurrentUser(c.GetHeader("X-Auth-Token"))

	if user == nil {
		resp.LogInvalidUser(c)
		return
	}

	tags := strings.Split(p.Tags, ",")
	post, err := CreatePost(p.Title, p.Content, user.Id, p.NodeId, p.Fee, tags)
	if err != nil {
		resp.LogError(c, err)
		return
	}

	item := repo.ItemResp[Post, PostResp](post)

	resp.OK(c, gin.H{"post": item})
}
