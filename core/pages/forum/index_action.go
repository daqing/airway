package forum

import (
	"fmt"
	"time"

	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type PostItemIndex struct {
	Id    int64
	Title string
	Url   string
	Date  string
}

type NodeItem struct {
	Id   int64
	Name string
	URL  string
}

func IndexAction(c *gin.Context) {
	posts, err := post_api.Posts(
		[]string{"id", "title", "custom_path"},
		"forum", // TODO: define constant
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
		url := fmt.Sprintf("/forum/post/%d", post.Id)

		if len(post.CustomPath) > 0 {
			url = fmt.Sprintf("/forum/post/%s", post.CustomPath)
		}

		postsShow = append(postsShow,
			&PostItemIndex{
				Id:    post.Id,
				Title: post.Title,
				Url:   url,
				Date:  post.CreatedAt.Format("2006-01-02 15:04"),
			},
		)
	}

	nodes, err := repo.Find[node_api.Node](
		[]string{"id", "name", "key"},
		[]repo.KVPair{
			repo.KV("place", "forum"),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	rootPath := utils.PathPrefix("forum")

	nodeItems := []*NodeItem{}

	for _, node := range nodes {
		nodeItems = append(nodeItems,
			&NodeItem{
				Name: node.Name,
				URL:  "/forum/node/" + node.Key,
			})
	}

	token, _ := c.Cookie("user_api_token")

	data := map[string]any{
		"Title":    ForumTitle(),
		"Tagline":  ForumTagline(),
		"Year":     time.Now().Year(),
		"RootPath": rootPath,
		"Nodes":    nodeItems,
		"Posts":    postsShow,
		"Session":  SessionData(token),
	}

	page_resp.Page(c, "core", "forum!", "index", data)
}
