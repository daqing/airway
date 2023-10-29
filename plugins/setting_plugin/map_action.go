package setting_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func MapAction(c *gin.Context) {
	settings, err := repo.Find[Setting]([]string{
		"id", "key", "val",
	}, []repo.KVPair{})

	if err != nil {
		resp.LogError(c, err)
		return
	}

	var mapping = make(map[string]string)

	for _, item := range settings {
		mapping[item.Key] = item.Val
	}

	resp.OK(c, mapping)
}
