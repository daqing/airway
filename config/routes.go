package config

import (
	"github.com/daqing/airway/app/routes/hello"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	hello.Routes(v1)
}
