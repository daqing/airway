package setting_plugin

import (
	"fmt"

	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
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

func UpdateAction(c *gin.Context) {
	var p UpdateParams

	if err := c.BindJSON(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	for _, item := range p.Data {
		if !UpdateSetting(item.Id, item.Key, item.Val) {
			utils.LogError(c,
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
