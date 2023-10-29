package setting_api

import (
	"fmt"

	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

type Item struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

type CreateParams struct {
	Data []Item `json:"data"`
}

func AdminCreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.BindJSON(&p); err != nil {
		resp.LogError(c, err)
		return
	}

	if len(p.Data) == 0 {
		resp.Error(c, fmt.Errorf("no data provided"))
		return
	}

	if !user_api.CheckAdmin(c.GetHeader("X-Auth-Token")) {
		resp.LogInvalidAdmin(c)
		return
	}

	for _, item := range p.Data {
		if _, err := CreateSetting(item.Key, item.Val); err != nil {
			resp.LogError(c, err)
			return
		}
	}

	resp.OK(c, gin.H{})
}
