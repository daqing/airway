package hello_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	list, err := repo.Find[Hello]([]string{
		"id", "name", "age", "created_at", "updated_at",
	}, []repo.KeyValueField{})

	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"list": list})
}
