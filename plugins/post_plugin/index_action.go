package post_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	posts, err := repo.Find[Post]([]string{
		"id", "user_id", "node_id", "title", "content",
	}, []repo.KeyValueField{})

	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"list": posts})
}
