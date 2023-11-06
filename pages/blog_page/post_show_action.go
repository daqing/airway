package blog_page

import (
	"strconv"
	"time"

	"github.com/daqing/airway/api/post_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func PostShowAction(c *gin.Context) {
	var segment = c.Param("segment")

	var where []repo.KVPair

	id, err := strconv.Atoi(segment)
	if err != nil {
		// segment is not numeric id
		where = []repo.KVPair{repo.KV("custom_path", segment)}
	} else {
		where = []repo.KVPair{repo.KV("id", id)}
	}

	post, err := repo.FindRow[post_api.Post](
		[]string{"id", "title", "content"},
		where,
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Title":        BlogTitle(),
		"Tagline":      BlogTagline(),
		"Year":         time.Now().Year(),
		"BlogRootPath": utils.PathPrefix("blog"),
		"PostDate":     post.CreatedAt.Format("2006-01-02"),
		"Post":         post,
	}

	page_resp.Page(c, "blog", "post_show", data)

}
