package user_plugin

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/user")
	{
		g.GET("/index", IndexAction)
		g.POST("/create", CreateAction)
		g.POST("/login", LoginAction)
		g.POST("/login_admin", LoginAdminAction)
	}
}
