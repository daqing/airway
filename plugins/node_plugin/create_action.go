package node_plugin

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.BindJSON(&p); err != nil {
		utils.LogError(c, err)
		return
	}

	node, err := CreateNode(p.Name, p.Key)
	if err != nil {
		utils.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"node": node})
}
