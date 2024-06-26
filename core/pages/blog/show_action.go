package blog

import (
	"bytes"
	"html/template"
	"strconv"
	"time"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/core/api/menu_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
	"github.com/yuin/goldmark"
)

func ShowAction(c *gin.Context) {
	segment := c.Param("id")

	var where []repo.KVPair

	id, err := strconv.Atoi(segment)
	if err != nil {
		// segment is not numeric id
		where = []repo.KVPair{repo.KV("custom_path", segment)}
	} else {
		where = []repo.KVPair{repo.KV("id", id)}
	}

	post, err := repo.FindOne[models.Post](
		[]string{"id", "title", "content", "created_at"},
		where,
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	menus, err := menu_api.MenuPlace(
		[]string{"name", "url"},
		"blog",
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	data := map[string]any{
		"Title":    BlogTitle(),
		"Tagline":  BlogTagline(),
		"Year":     time.Now().Year(),
		"RootPath": utils.PathPrefix("blog"),
		"PostDate": post.CreatedAt.Format("2006-01-02"),
		"Post":     post,
		"Menus":    menus,
	}

	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(post.Content), &buf); err != nil {
		page_resp.Error(c, err)
		return
	}

	data["ContentHTML"] = template.HTML(buf.String())

	page_resp.Page(c, "core", "blog!", "show", data)

}
