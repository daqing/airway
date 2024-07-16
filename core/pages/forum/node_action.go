package forum

import (
	"fmt"
	"time"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/app/repos/post_repo"
	"github.com/daqing/airway/core/api/media_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func NodeAction(c *gin.Context) {
	nodeKey := c.Param("key")

	node, err := sql_orm.FindOne[models.Node](
		[]string{"id", "name"},
		[]sql_orm.KVPair{
			sql_orm.KV("key", nodeKey),
		},
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	posts, err := sql_orm.Find[models.Post](
		[]string{"id", "title", "user_id", "created_at"},
		[]sql_orm.KVPair{
			sql_orm.KV("node_id", node.ID),
			sql_orm.KV("place", "forum"),
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
				AvatarURL: media_api.AssetHostPath(post_repo.PostUserAvatar(post)),
			},
		)
	}

	rootPath := utils.PathPrefix("forum")

	nodes, err := sql_orm.Find[models.Node](
		[]string{"id", "name", "key"},
		[]sql_orm.KVPair{
			sql_orm.KV("place", "forum"),
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
