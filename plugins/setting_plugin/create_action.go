package setting_plugin

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type Item struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

type CreateParams struct {
	Data []Item `json:"data"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.BindJSON(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	for _, item := range p.Data {
		if _, err := CreateSetting(item.Key, item.Val); err != nil {
			utils.LogError(c, err)
			return
		}
	}

	resp.OK(c, gin.H{})
}
