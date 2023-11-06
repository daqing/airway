package blog_page

import (
	"time"

	"github.com/daqing/airway/api/post_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	posts, err := post_api.Posts("id DESC", 0, 50)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Title":        BlogTitle(),
		"Tagline":      BlogTagline(),
		"Year":         time.Now().Year(),
		"BlogRootPath": utils.PathPrefix("blog"),
		"Posts":        posts,
	}

	page_resp.Page(c, "blog", "index", data)
}
