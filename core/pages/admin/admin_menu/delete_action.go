package admin_menu

import (
	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/gin-gonic/gin"
)

func DeleteAction(c *gin.Context) {
	id := c.Query("id")

	err := sql_orm.Delete[models.Menu](
		[]sql_orm.KVPair{
			sql_orm.KV("id", id),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	page_resp.Redirect(c, "/admin/menu")
}
