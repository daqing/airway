package up_page

import "github.com/gin-gonic/gin"

func Routes(r *gin.Engine) {
	g := r.Group("/up")
	{
		g.GET("", IndexAction)
	}
}
