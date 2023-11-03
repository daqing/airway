package sign_out_page

import "github.com/gin-gonic/gin"

func Routes(r *gin.Engine) {
	g := r.Group("/sign_out")
	{
		g.GET("/index", IndexAction)
	}
}
