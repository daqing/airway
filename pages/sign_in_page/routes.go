package sign_in_page

import "github.com/gin-gonic/gin"

func Routes(r *gin.Engine) {
	g := r.Group("/sign_in")
	{
		g.GET("/index", IndexAction)
	}
}
