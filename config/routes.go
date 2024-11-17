package config

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/api/up_api"
	"github.com/daqing/airway/app/api/user_api"
	"github.com/daqing/airway/app/websocket"
)

func Routes(r *gin.Engine) {
	websocketRoutes(r)

	apiRoutes(r)
}

func apiRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	up_api.Routes(v1)

	user_api.Routes(v1)
}

func websocketRoutes(r *gin.Engine) {
	r.GET("/ws", websocket.Conn)
}
