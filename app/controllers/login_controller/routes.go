package login_controller

import "github.com/gin-gonic/gin"

func Routes(r *gin.Engine) {
	g := r.Group("/login")
	{
		g.GET("/index", IndexAction)
	}
}
