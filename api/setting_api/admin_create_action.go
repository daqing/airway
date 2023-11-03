package setting_api

import (
	"fmt"

	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/api_resp"

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
		api_resp.LogError(c, err)
		return
	}

	if len(p.Data) == 0 {
		api_resp.Error(c, fmt.Errorf("no data provided"))
		return
	}

	if !user_api.CheckAdmin(c.GetHeader("X-Auth-Token")) {
		api_resp.LogInvalidAdmin(c)
		return
	}

	for _, item := range p.Data {
		if _, err := CreateSetting(item.Key, item.Val); err != nil {
			api_resp.LogError(c, err)
			return
		}
	}

	api_resp.OK(c, gin.H{})
}
