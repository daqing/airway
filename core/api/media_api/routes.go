package media_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/media")
	{
		g.POST("/upload", UploadAction)
	}
}
