package config

import (
	"time"

	"github.com/daqing/airway/app/middleware"
	"github.com/daqing/airway/app/modules/identity"
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/api/health_api"
	"github.com/daqing/airway/app/api/session_api"
	"github.com/daqing/airway/app/api/storage_api"
	"github.com/daqing/airway/app/websocket"
	"github.com/daqing/airway/lib/repo"
)

func Routes(r *gin.Engine) {
	health_api.Routes(r)

	websocketRoutes(r)
	apiGroupRoutes(r)
}

func apiGroupRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	session_api.Routes(v1)

	// Login is public and session endpoints perform their own authentication.
	// Every business API is mounted below an authenticated group by default.
	if db, ok := repo.CurrentDBOK(); ok {
		service := identity.NewService(db, 12*time.Hour)
		protected := v1.Group("")
		protected.Use(middleware.Authenticate(service))
		protected.Use(middleware.RequireSuperAdmin())
		storage_api.Routes(protected)
	}
}

func websocketRoutes(r *gin.Engine) {
	r.GET("/ws", websocket.Conn)
	r.POST("/ws/publish", websocket.Publish)
}
