package config

import (
	"github.com/daqing/airway/api/checkin_api"
	"github.com/daqing/airway/api/comment_api"
	"github.com/daqing/airway/api/hello_api"
	"github.com/daqing/airway/api/node_api"
	"github.com/daqing/airway/api/post_api"
	"github.com/daqing/airway/api/setting_api"
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/app/controllers/home_controller"
	"github.com/daqing/airway/pages/demo_page"
	"github.com/daqing/airway/pages/up_page"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	r.GET("/", home_controller.IndexAction)

	demo_page.Routes(r)
	up_page.Routes(r)

	v1 := r.Group("/api/v1")

	checkin_api.Routes(v1)
	comment_api.Routes(v1)
	hello_api.Routes(v1)
	node_api.Routes(v1)
	post_api.Routes(v1)
	setting_api.Routes(v1)
	user_api.Routes(v1)
}
