package up_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/up")
	{
		g.GET("", IndexAction)
	}
}
