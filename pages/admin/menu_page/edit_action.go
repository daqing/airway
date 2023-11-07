package menu_page

import (
	"github.com/daqing/airway/api/menu_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func EditAction(c *gin.Context) {
	id := utils.TrimFull(c.Query("id"))

	if len(id) == 0 {
		page_resp.Redirect(c, "/admin/menu")
	}

	menu, err := menu_api.FindBy("id", id)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Menu": menu,
	}

	page_resp.Page(c, "admin/menu", "edit", data)
}
