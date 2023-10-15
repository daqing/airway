package config

import (
	"github.com/daqing/airway/plugins/hello_plugin"
	"github.com/daqing/airway/plugins/user_plugin"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	user_plugin.Routes(v1)
	hello_plugin.Routes(v1)
}
