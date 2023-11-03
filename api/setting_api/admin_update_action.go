package setting_api

import (
	"fmt"

	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/api_resp"

	"github.com/gin-gonic/gin"
)

type UpdateItem struct {
	Id  int64  `json:"id"`
	Key string `json:"key"`
	Val string `json:"val"`
}

type UpdateParams struct {
	Data []UpdateItem `json:"data"`
}

func AdminUpdateAction(c *gin.Context) {
	var p UpdateParams

	if err := c.BindJSON(&p); err != nil {
		api_resp.LogError(c, err)
		return
	}

	if !user_api.CheckAdmin(c.GetHeader("X-Auth-Token")) {
		api_resp.LogInvalidAdmin(c)
		return
	}

	for _, item := range p.Data {
		if !UpdateSetting(item.Id, item.Key, item.Val) {
			api_resp.LogError(c,
				fmt.Errorf(
					"update setting %d failed: %s, %s",
					item.Id,
					item.Key,
					item.Val,
				),
			)
		}
	}

	api_resp.OK(c, gin.H{})
}
