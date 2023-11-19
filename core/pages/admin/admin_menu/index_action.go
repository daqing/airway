package admin_menu

import (
	"github.com/daqing/airway/core/api/menu_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	menus, err := menu_api.Menus(
		[]string{"id", "name", "url", "place"},
		"id DESC",
		0,
		50,
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Menus": menus,
	}

	page_resp.Page(c, "core", "admin/menu", "index", data)
}
