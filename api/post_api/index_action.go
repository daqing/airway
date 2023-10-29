package post_api

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	list, err := repo.ListResp[Post, PostResp]()
	if err != nil {
		resp.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"list": list})
}
