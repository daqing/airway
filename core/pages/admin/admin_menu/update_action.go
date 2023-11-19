package admin_menu

import (
	"fmt"

	"github.com/daqing/airway/core/api/menu_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type UpdateParams struct {
	Id    int64  `form:"id"`
	Name  string `form:"name"`
	URL   string `form:"url"`
	Place string `form:"place"`
}

func UpdateAction(c *gin.Context) {
	var p UpdateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	id := p.Id
	name := utils.TrimFull(p.Name)
	url := utils.TrimFull(p.URL)
	place := utils.TrimFull(p.Place)

	if id <= 0 {
		page_resp.Error(c, fmt.Errorf("id must be greater than 0"))
		return
	}

	if len(name) == 0 || len(url) == 0 || len(place) == 0 {
		page_resp.Error(c, fmt.Errorf("name or url or place must not be empty"))
		return
	}

	ok := menu_api.UpdateMenu(id, name, url, place)
	if !ok {
		page_resp.Error(c, fmt.Errorf("update menu failed"))
		return
	}

	page_resp.Redirect(c, "/admin/menu")
}
