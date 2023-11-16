package ext_date_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/date")
	{
		g.GET("", IndexAction)
	}
}
