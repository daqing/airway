package blog_page

import "github.com/gin-gonic/gin"

func Routes(r *gin.Engine) {
	g := r.Group("/blog")
	{
		g.GET("", IndexAction)
		g.GET("/post/:segment", PostShowAction)
	}
}
