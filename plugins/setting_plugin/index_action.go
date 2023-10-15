package setting_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	settings, err := repo.Find[Setting]([]string{
		"id", "key", "val",
	}, []repo.KeyValueField{})

	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"list": settings})
}
