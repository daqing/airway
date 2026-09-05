package health_api

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	r.GET("/health", HealthAction)
}
