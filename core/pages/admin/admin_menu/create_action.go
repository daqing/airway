package admin_menu

import (
	"fmt"

	"github.com/daqing/airway/core/api/menu_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	Name  string `form:"name"`
	URL   string `form:"url"`
	Place string `form:"place"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	name := utils.TrimFull(p.Name)
	url := utils.TrimFull(p.URL)
	place := utils.TrimFull(p.Place)

	if len(name) == 0 || len(url) == 0 || len(place) == 0 {
		page_resp.Error(c, fmt.Errorf("name or url or place must not be empty"))
		return
	}

	_, err := menu_api.CreateMenu(name, url, place)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	page_resp.Redirect(c, "/admin/menu")
}
