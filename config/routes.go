package config

import (
	"github.com/daqing/airway/core/api/checkin_api"
	"github.com/daqing/airway/core/api/comment_api"
	"github.com/daqing/airway/core/api/hello_api"
	"github.com/daqing/airway/core/api/node_api"
	"github.com/daqing/airway/core/api/post_api"
	"github.com/daqing/airway/core/api/setting_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/core/pages/admin"
	"github.com/daqing/airway/core/pages/blog"
	"github.com/daqing/airway/core/pages/forum"
	"github.com/daqing/airway/core/pages/home_page"
	"github.com/daqing/airway/core/pages/session_page"
	"github.com/daqing/airway/core/pages/up_page"
	"github.com/daqing/airway/ext/api/ext_date_api"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	r.GET("/", home_page.IndexAction)

	coreRoutes(r)
	extRoutes(r)
}

func coreRoutes(r *gin.Engine) {
	up_page.Routes(r)
	session_page.Routes(r)

	blog.Routes(r)
	forum.Routes(r)

	admin.Routes(r)

	coreAPIRoutes(r)
}

func coreAPIRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	checkin_api.Routes(v1)
	comment_api.Routes(v1)
	hello_api.Routes(v1)
	node_api.Routes(v1)
	post_api.Routes(v1)
	setting_api.Routes(v1)
	user_api.Routes(v1)
}

func extRoutes(r *gin.Engine) {
	extAPIRoutes(r)
}

func extAPIRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	ext_date_api.Routes(v1)
}
