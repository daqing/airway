package storage_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/storage")
	{
		g.POST("/upload", UploadAction)
	}
}
