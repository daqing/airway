package forum

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	g := r.Group("/forum")
	{
		g.GET("", IndexAction)

		g.GET("/post/:id", ShowAction)
		g.GET("/node/:key", NodeAction)

		g.GET("/post/new", NewAction)
		g.POST("/post/create", CreateAction)

		g.GET("/settings", SettingsAction)
		g.POST("/settings/update_avatar", UpdateAvatarAction)

		g.POST("/comment/create", CreateCommentAction)
	}
}
