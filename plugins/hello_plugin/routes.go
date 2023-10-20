package hello_plugin

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/hello")
	{
		g.GET("/ping", PingAction)
	}
}
