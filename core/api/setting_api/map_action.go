package setting_api

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/gin-gonic/gin"
)

func MapAction(c *gin.Context) {
	settings, err := sql_orm.Find[models.Setting]([]string{
		"id", "key", "val",
	}, []sql_orm.KVPair{})

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
