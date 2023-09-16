package hello

import (
	"github.com/daqing/airway/app/controllers/hello_controller"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/hello")
	{
		g.GET("/", hello_controller.Index)
	}
}
