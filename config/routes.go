package config

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/api/health_api"
	"github.com/daqing/airway/app/api/home_api"
	"github.com/daqing/airway/app/api/storage_api"
	"github.com/daqing/airway/app/websocket"
)

// Routes registers every route — public and internal — at the root paths. This
// is the full router used when the app is served without a URL_PREFIX.
func Routes(r *gin.Engine) {
	PublicRoutes(r)
	HealthRoutes(r)
}

// PublicRoutes registers the user-facing routes: the home page, the WebSocket,
// and the API. When a URL_PREFIX is configured these answer only under the
// prefix; see App.Handler.
func PublicRoutes(r *gin.Engine) {
	r.GET("/", home_api.IndexAction)

	websocketRoutes(r)
	apiGroupRoutes(r)
}

// HealthRoutes registers the internal health-check route. It stays reachable at
// the unprefixed root (for load-balancer probes) even when the public routes
// are served under a URL_PREFIX.
func HealthRoutes(r *gin.Engine) {
	health_api.Routes(r)
}

func apiGroupRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		storage_api.Routes(v1)
	}
}

func websocketRoutes(r *gin.Engine) {
	r.GET("/ws", websocket.Conn)
	r.POST("/ws/publish", websocket.Publish)
}
