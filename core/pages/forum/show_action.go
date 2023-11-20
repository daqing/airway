package forum

import (
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
)

type CommentItem struct {
	Nickname    string
	AvatarURL   string
	ContentHTML template.HTML
	CreatedAt   string
}

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

	comments, err := post.Comments()
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	commentItems := []*CommentItem{}

	for _, comment := range comments {
		user := comment.User()
		if user == nil {
			continue
		}

		commentItems = append(commentItems,
			&CommentItem{
				Nickname:    user.Nickname,
				AvatarURL:   media_api.AssetHostPath(user.Avatar),
				ContentHTML: utils.RenderMarkdown(comment.Content),
				CreatedAt:   utils.TimeAgo(comment.CreatedAt),
			},
		)
	}

	data := map[string]any{
		"Title":       ForumTitle(),
		"Tagline":     ForumTagline(),
		"Year":        time.Now().Year(),
		"RootPath":    rootPath,
		"Nodes":       nodeItems,
		"Post":        post,
		"PostUser":    postUser,
		"PostNode":    postNode,
		"Session":     SessionData(currentUser),
		"AvatarURL":   media_api.AssetHostPath(post.UserAvatar()),
		"Comments":    commentItems,
		"HasComments": len(commentItems) > 0,
	}

	data["ContentHTML"] = utils.RenderMarkdown(post.Content)

	page_resp.Page(c, "core", "forum!", "show", data)
}
