package blog_page

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	data := map[string]any{
		"Title":   BlogTitle(),
		"Tagline": BlogTagline(),
	}
	resp.Page(c, "blog", "index", data)
}
