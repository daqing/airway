package blog_page

import (
	"time"

	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	data := map[string]any{
		"Title":   BlogTitle(),
		"Tagline": BlogTagline(),
		"Year":    time.Now().Year(),
	}

	resp.Page(c, "blog", "index", data)
}
