package setting_plugin

import "github.com/gin-gonic/gin"

func Routes(r *gin.RouterGroup) {
	g := r.Group("/setting")
	{
		g.GET("/index", IndexAction)
		g.GET("/map", MapAction)
		g.POST("/create", CreateAction)
		g.POST("/update", UpdateAction)
	}
}
