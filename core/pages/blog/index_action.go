package blog

import (
	"fmt"
	"time"

	"github.com/daqing/airway/core/api/menu_api"
	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type PostItemIndex struct {
	Id    int64
	Title string
	Url   string
	Date  string
}

func IndexAction(c *gin.Context) {
	posts, err := post_api.Posts(
		[]string{"id", "title", "custom_path"},
		"blog", // TODO: define constant
		"id DESC",
		0,
		50,
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	postsShow := []*PostItemIndex{}

	for _, post := range posts {
		url := fmt.Sprintf("/blog/post/%d", post.Id)

		if len(post.CustomPath) > 0 {
			url = fmt.Sprintf("/blog/post/%s", post.CustomPath)
		}

		postsShow = append(postsShow,
			&PostItemIndex{
				Id:    post.Id,
				Title: post.Title,
				Url:   url,
				Date:  post.CreatedAt.Format("2006-01-02"),
			},
		)
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
		"Title":        BlogTitle(),
		"Tagline":      BlogTagline(),
		"Year":         time.Now().Year(),
		"BlogRootPath": utils.PathPrefix("blog"),
		"Menus":        menus,
		"Posts":        postsShow,
	}

	page_resp.Expand(c, "core", "blog", "index", data)
}
