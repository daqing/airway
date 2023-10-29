package node_plugin

import (
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/plugins/user_plugin"
	"github.com/gin-gonic/gin"
)

type AdminCreateParams struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

func AdminCreateAction(c *gin.Context) {
	var p AdminCreateParams

	if err := c.BindJSON(&p); err != nil {
		resp.LogError(c, err)
		return
	}

	if !user_plugin.CheckAdmin(c.GetHeader("X-Auth-Token")) {
		resp.LogInvalidAdmin(c)
		return
	}

	node, err := CreateNode(p.Name, p.Key)
	if err != nil {
		resp.LogError(c, err)
		return
	}

	resp.OK(c, gin.H{"node": repo.ItemResp[Node, NodeResp](node)})
}
