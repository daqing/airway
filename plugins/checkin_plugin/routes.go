package checkin_plugin

import "github.com/gin-gonic/gin"

func Routes(r *gin.RouterGroup) {
	g := r.Group("/checkin")
	{
		g.POST("/create", CreateAction)
	}
}
