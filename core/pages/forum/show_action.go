package forum

import (
	"html/template"
	"strconv"
	"time"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/app/repos/comment_repo"
	"github.com/daqing/airway/app/repos/post_repo"
	"github.com/daqing/airway/core/api/media_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/sql_orm"
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

	var where []sql_orm.KVPair

	id, err := strconv.Atoi(segment)
	if err != nil {
		// segment is not numeric id
		where = []sql_orm.KVPair{sql_orm.KV("custom_path", segment)}
	} else {
		where = []sql_orm.KVPair{sql_orm.KV("id", id)}
	}

	post, err := sql_orm.FindOne[models.Post](
		[]string{"id", "title", "content", "user_id", "node_id", "created_at"},
		where,
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

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

	postUser := post_repo.PostUser(post)
	if postUser == nil {
		// user not found
		page_resp.Redirect(c, "/forum")
		return
	}

	postNode := post_repo.PostNode(post)
	if postNode == nil {
		postNode = &models.Node{}
	}

	comments, err := post_repo.PostComments(post)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	commentItems := []*CommentItem{}

	for _, comment := range comments {
		user := comment_repo.CommentUser(comment)
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
		"AvatarURL":   media_api.AssetHostPath(post_repo.PostUserAvatar(post)),
		"Comments":    commentItems,
		"HasComments": len(commentItems) > 0,
	}

	data["ContentHTML"] = utils.RenderMarkdown(post.Content)

	page_resp.Page(c, "core", "forum!", "show", data)
}
