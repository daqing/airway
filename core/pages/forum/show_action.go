package forum

import (
	"bytes"
	"html/template"
	"strconv"
	"time"

	"github.com/daqing/airway/core/api/media_api"
	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/core/api/user_api"
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

	post, err := repo.FindRow[post_api.Post](
		[]string{"id", "title", "content", "user_id", "node_id"},
		where,
	)

	if err != nil {
		page_resp.Error(c, err)
		return
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

	token, _ := utils.CookieToken(c)
	currentUser := user_api.CurrentUser(token)

	postUser := post.User()
	if postUser == nil {
		// user not found
		page_resp.Redirect(c, "/forum")
		return
	}

	postNode := post.Node()
	if postNode == nil {
		postNode = &node_api.Node{}
	}

	data := map[string]any{
		"Title":     ForumTitle(),
		"Tagline":   ForumTagline(),
		"Year":      time.Now().Year(),
		"RootPath":  rootPath,
		"Nodes":     nodeItems,
		"Post":      post,
		"PostUser":  postUser,
		"PostNode":  postNode,
		"Session":   SessionData(currentUser),
		"AvatarURL": media_api.AssetHostPath(post.UserAvatar()),
	}

	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(post.Content), &buf); err != nil {
		page_resp.Error(c, err)
		return
	}

	data["ContentHTML"] = template.HTML(buf.String())

	page_resp.Page(c, "core", "forum!", "show", data)
}
