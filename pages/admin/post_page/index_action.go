package post_page

import (
	"github.com/daqing/airway/api/post_api"
	"github.com/daqing/airway/lib/page_resp"

	"github.com/gin-gonic/gin"
)

type IndexParams struct {
	Page int `form:"page"`
}

func IndexAction(c *gin.Context) {
	var p IndexParams

	if err := c.Bind(&p); err != nil {
		c.AbortWithError(500, err)
		return
	}

	posts, err := post_api.Posts("id DESC", p.Page, 50)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Posts": posts,
	}

	page_resp.Page(c, "admin/post", "index", data)
}
