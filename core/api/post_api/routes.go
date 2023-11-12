package post_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/post")
	{
		g.GET("/index", IndexAction)
		g.GET("/show", ShowAction)
		g.POST("/create", CreateAction)

		g.POST("/toggle/like", ToggleLikeAction)
		g.POST("/toggle/favorite", ToggleFavoriteAction)
	}
}
