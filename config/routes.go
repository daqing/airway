package config

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/api/date_api"
	"github.com/daqing/airway/app/api/user_api"
)

func Routes(r *gin.Engine) {
	appRoutes(r)
}

func appRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	user_api.Routes(v1)

	date_api.Routes(v1)
}
