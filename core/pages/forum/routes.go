package forum

import (
	"github.com/daqing/airway/core/pages/forum/forum_home"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	g := r.Group("/forum")
	{
		g.GET("", forum_home.IndexAction)
	}
}
