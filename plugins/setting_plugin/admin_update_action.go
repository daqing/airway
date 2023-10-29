package setting_plugin

import (
	"fmt"

	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/plugins/user_plugin"
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
		resp.LogError(c, err)
		return
	}

	if !user_plugin.CheckAdmin(c.GetHeader("X-Auth-Token")) {
		resp.LogInvalidAdmin(c)
		return
	}

	for _, item := range p.Data {
		if !UpdateSetting(item.Id, item.Key, item.Val) {
			resp.LogError(c,
				fmt.Errorf(
					"update setting %d failed: %s, %s",
					item.Id,
					item.Key,
					item.Val,
				),
			)
		}
	}

	resp.OK(c, gin.H{})
}
