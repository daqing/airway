package user_page

import (
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/gin-gonic/gin"
)

type IndexParams struct {
	Page int `form:"page"`
}

func IndexAction(c *gin.Context) {
	var p IndexParams

	if err := c.Bind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	users, err := user_api.Users(
		[]string{"id", "nickname", "username", "role", "api_token", "balance"},
		"id DESC",
		p.Page,
		50,
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Users": users,
	}

	page_resp.Page(c, "admin/user", "index", data)
}
