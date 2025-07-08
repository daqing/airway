package asset_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	g := r.Group("/asset")
	{
		g.GET("/*path", ServeAction)
	}
}
