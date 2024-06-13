package forum

import (
	"fmt"
	"time"

	"github.com/daqing/airway/core/api/media_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/models"
	"github.com/gin-gonic/gin"
)

func NodeAction(c *gin.Context) {
	nodeKey := c.Param("key")

	node, err := repo.FindRow[models.Node](
		[]string{"id", "name"},
		[]repo.KVPair{
			repo.KV("key", nodeKey),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	posts, err := repo.Find[models.Post](
		[]string{"id", "title", "user_id"},
		[]repo.KVPair{
			repo.KV("node_id", node.ID),
			repo.KV("place", "forum"),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	postsShow := []*PostItem{}

	for _, post := range posts {
		url := fmt.Sprintf("/forum/post/%d", post.ID)

		if len(post.CustomPath) > 0 {
			url = fmt.Sprintf("/forum/post/%s", post.CustomPath)
		}

		postsShow = append(postsShow,
			&PostItem{
				Id:        post.ID,
				Title:     post.Title,
				Url:       url,
				TimeAgo:   utils.TimeAgo(post.CreatedAt),
				UserName:  user_api.Nickname(post.UserId),
				AvatarURL: media_api.AssetHostPath(post.UserAvatar()),
			},
		)
	}

	rootPath := utils.PathPrefix("forum")

	nodes, err := repo.Find[models.Node](
		[]string{"id", "name", "key"},
		[]repo.KVPair{
			repo.KV("place", "forum"),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	nodeItems := []*NodeItem{}

	for _, node := range nodes {
		nodeItems = append(nodeItems,
			&NodeItem{
				Name: node.Name,
				URL:  "/forum/node/" + node.Key,
			})
	}

	token, _ := utils.CookieToken(c)
	currentUser := user_api.CurrentUser(token)

	data := map[string]any{
		"Title":    ForumTitle(),
		"Tagline":  ForumTagline(),
		"Year":     time.Now().Year(),
		"RootPath": rootPath,
		"Node":     node,
		"Nodes":    nodeItems,
		"Posts":    postsShow,
		"Session":  SessionData(currentUser),
	}

	page_resp.Page(c, "core", "forum!", "node", data)
}
