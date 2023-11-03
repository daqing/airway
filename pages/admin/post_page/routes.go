package post_page

import "github.com/gin-gonic/gin"

func Routes(r *gin.RouterGroup) {
	g := r.Group("/post")
	{
		g.GET("", IndexAction)
	}
}
