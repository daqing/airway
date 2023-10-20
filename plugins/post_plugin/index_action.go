package post_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	list, err := repo.ListResp[Post, PostResp]()
	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"list": list})
}
