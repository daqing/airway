package node_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/node")
	{
		g.GET("/index", IndexAction)

	}
}
