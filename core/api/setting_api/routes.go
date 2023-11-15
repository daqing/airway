package setting_api

import "github.com/gin-gonic/gin"

func Routes(r *gin.RouterGroup) {
	g := r.Group("/setting")
	{
		g.GET("/map", MapAction)
	}
}
