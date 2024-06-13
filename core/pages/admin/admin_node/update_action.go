package admin_node

import (
	"fmt"

	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/models"
	"github.com/gin-gonic/gin"
)

type UpdateParams struct {
	Id    uint   `form:"id"`
	Name  string `form:"name"`
	Key   string `form:"key"`
	Place string `form:"place"`
}

func UpdateAction(c *gin.Context) {
	var p UpdateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	ok := repo.UpdateFields[models.Node](p.Id,
		[]repo.KVPair{
			repo.KV("name", p.Name),
			repo.KV("key", p.Key),
			repo.KV("place", p.Place),
		},
	)

	if !ok {
		page_resp.Error(c, fmt.Errorf("update failed"))
		return
	}

	page_resp.Redirect(c, "/admin/node")

}
