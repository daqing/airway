package setting_api

import (
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/pg_repo"
	"github.com/gin-gonic/gin"
)

func MapAction(c *gin.Context) {
	settings, err := pg_repo.Find[Setting]([]string{
		"id", "key", "val",
	}, []pg_repo.KVPair{})

	if err != nil {
		api_resp.LogError(c, err)
		return
	}

	var mapping = make(map[string]string)

	for _, item := range settings {
		mapping[item.Key] = item.Val
	}

	api_resp.OK(c, mapping)
}
