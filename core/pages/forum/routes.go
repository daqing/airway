package forum

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	g := r.Group("/forum")
	{
		g.GET("", IndexAction)

		g.GET("/post/:id", ShowAction)
	}
}
