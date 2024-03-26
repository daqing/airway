package admin_menu

import (
	"github.com/daqing/airway/core/api/menu_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/pg_repo"
	"github.com/gin-gonic/gin"
)

func DeleteAction(c *gin.Context) {
	id := c.Query("id")

	err := pg_repo.Delete[menu_api.Menu](
		[]pg_repo.KVPair{
			pg_repo.KV("id", id),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	page_resp.Redirect(c, "/admin/menu")
}
