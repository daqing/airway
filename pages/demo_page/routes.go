package demo_page

import "github.com/gin-gonic/gin"

func Routes(r *gin.Engine) {
	g := r.Group("/demo")
	{
		g.GET("/index", IndexAction)
	}
}
