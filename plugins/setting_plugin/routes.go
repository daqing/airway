package setting_plugin

import "github.com/gin-gonic/gin"

func Routes(r *gin.RouterGroup) {
	g := r.Group("/setting")
	{
		g.GET("/map", MapAction)
	}

	admin := g.Group("/admin")
	{
		admin.GET("/index", AdminIndexAction)
		admin.POST("/create", AdminCreateAction)
		admin.POST("/update", AdminUpdateAction)
	}
}
