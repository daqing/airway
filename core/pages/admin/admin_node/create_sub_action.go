package admin_node

import (
	"fmt"

	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/models"
	"github.com/gin-gonic/gin"
)

type CreateSubParams struct {
	ParentId uint   `form:"parent_id"`
	Name     string `form:"name"`
	Key      string `form:"key"`
	Place    string `form:"place"`
}

func CreateSubAction(c *gin.Context) {
	var p CreateSubParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	parentNode, err := repo.FindRow[models.Node](
		[]string{"id", "level"},
		[]repo.KVPair{repo.KV("id", p.ParentId)},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	name := utils.TrimFull(p.Name)
	key := utils.TrimFull(p.Key)
	place := utils.TrimFull(p.Place)

	if len(name) == 0 || len(key) == 0 || len(place) == 0 {
		page_resp.Error(c, fmt.Errorf("name or key or place must be specified"))
		return
	}

	_, err = node_api.CreateNode(name, key, place, parentNode.ID, parentNode.Level+1)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	page_resp.Redirect(c, "/admin/node")
}
