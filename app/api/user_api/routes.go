package user_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/admin")
	{
		g.POST("/login", LoginAction)
	}
}
