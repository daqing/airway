package post_plugin

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	UserId  int64  `json:"user_id"`
	NodeId  int64  `json:"node_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.BindJSON(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	post, err := CreatePost(p.Title, p.Content, p.UserId, p.NodeId)
	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"post": post})
}
