package admin_node

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/gin-gonic/gin"
)

type UpdateParams struct {
	Id    models.IdType `form:"id"`
	Name  string        `form:"name"`
	Key   string        `form:"key"`
	Place string        `form:"place"`
}

func UpdateAction(c *gin.Context) {
	var p UpdateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	ok := sql_orm.UpdateFields[models.Node](p.Id,
		[]sql_orm.KVPair{
			sql_orm.KV("name", p.Name),
			sql_orm.KV("key", p.Key),
			sql_orm.KV("place", p.Place),
		},
	)

	if !ok {
		page_resp.Error(c, fmt.Errorf("update failed"))
		return
	}

	page_resp.Redirect(c, "/admin/node")

}
