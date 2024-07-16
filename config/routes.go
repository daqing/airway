package config

import (
	"github.com/daqing/airway/app/api/app_date_api"
	"github.com/daqing/airway/app/api/app_user_api"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	appRoutes(r)
}

func appRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	app_user_api.Routes(v1)

	app_date_api.Routes(v1)
}
