package config

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/api/page_api"
	"github.com/daqing/airway/app/api/up_api"
	"github.com/daqing/airway/app/websocket"
)

func Routes(r *gin.Engine) {
	up_api.Routes(r)

	websocketRoutes(r)
	apiGroupRoutes(r)
}

func apiGroupRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		page_api.Routes(v1)
	}
}

func websocketRoutes(r *gin.Engine) {
	r.GET("/ws", websocket.Conn)
	r.POST("/ws/publish", websocket.Publish)
}
