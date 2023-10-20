package comment_plugin

import "github.com/gin-gonic/gin"

func Routes(r *gin.RouterGroup) {
	g := r.Group("/comment")
	{
		g.POST("/create", CreateAction)
	}
}
