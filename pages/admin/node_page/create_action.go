package node_page

import (
	"fmt"

	"github.com/daqing/airway/api/node_api"
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	Name string `form:"name"`
	Key  string `form:"key"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	name := utils.TrimFull(p.Name)
	key := utils.TrimFull(p.Key)

	if len(name) == 0 || len(key) == 0 {
		page_resp.Error(c, fmt.Errorf("name or key must not be empty"))
		return
	}

	token, err := c.Cookie("user_api_token")
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	admin := user_api.CurrentAdmin(token)
	if admin == nil {
		page_resp.Error(c, fmt.Errorf("current user is not admin"))
		return
	}

	_, err = node_api.CreateNode(name, key)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	page_resp.Redirect(c, "/admin/node")
}