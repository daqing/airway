package config

import (
	"github.com/daqing/airway/plugins/checkin_plugin"
	"github.com/daqing/airway/plugins/comment_plugin"
	"github.com/daqing/airway/plugins/hello_plugin"
	"github.com/daqing/airway/plugins/node_plugin"
	"github.com/daqing/airway/plugins/post_plugin"
	"github.com/daqing/airway/plugins/setting_plugin"
	"github.com/daqing/airway/plugins/user_plugin"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	checkin_plugin.Routes(v1)
	comment_plugin.Routes(v1)
	hello_plugin.Routes(v1)
	node_plugin.Routes(v1)
	post_plugin.Routes(v1)
	setting_plugin.Routes(v1)
	user_plugin.Routes(v1)
}
