package blog_page

import (
	"time"

	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	data := map[string]any{
		"Title":        BlogTitle(),
		"Tagline":      BlogTagline(),
		"Year":         time.Now().Year(),
		"BlogRootPath": utils.PathPrefix("blog"),
	}

	page_resp.Page(c, "blog", "index", data)
}
