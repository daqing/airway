package hello_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/hello")
	{
		g.GET("/ping", PingAction)
	}
}
