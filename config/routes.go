package config

import (
	"github.com/daqing/airway/app/api/app_date_api"
	"github.com/daqing/airway/core/pages/home_page"
	"github.com/daqing/airway/core/pages/session_page"
	"github.com/daqing/airway/core/pages/up_page"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	r.GET("/", home_page.IndexAction)

	up_page.Routes(r)
	session_page.Routes(r)

	apiRoutes(r)
}

func apiRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	app_date_api.Routes(v1)
}
