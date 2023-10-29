package user_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/user")
	{
		g.POST("/create", CreateAction)
		g.POST("/login", LoginAction)
		g.POST("/login_admin", LoginAdminAction)

		g.POST("/toggle/follow", ToggleFollowAction)
	}

	admin := g.Group("/admin")
	{
		admin.GET("/index", AdminIndexAction)
	}
}
