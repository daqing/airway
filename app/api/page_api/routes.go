package page_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	r.GET("/ping", PingAction)
}
