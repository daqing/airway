package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/daqing/airway/config"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type App struct {
	r        *gin.Engine // Full router: public + internal routes at the root
	internal *gin.Engine // Internal-only router (health) for unprefixed requests
	name     string      // Application name
	port     string
	prefix   string // Public sub-path prefix ("" = serve at root)
}

func NewApp(name, port string) *App {
	router := newEngine()
	config.Routes(router)

	internal := newEngine()
	config.HealthRoutes(internal)

	return &App{
		r:        router,
		internal: internal,
		name:     name,
		port:     port,
		prefix:   utils.URLPrefix(),
	}
}

func newEngine() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORS())
	return router
}

// Handler returns the http.Handler that serves the app. When a public sub-path
// prefix is configured (e.g. "/airway") requests carrying that prefix are
// stripped before they reach the full router, so public routes stay registered
// at the root while the app answers under the prefix. Unprefixed requests are
// answered only by the internal router (the health check), so GET / no longer
// serves the public home page when the app lives under a prefix. With no prefix
// configured, every route answers at the root.
func (a *App) Handler() http.Handler {
	if a.prefix == "" {
		return a.r
	}

	return prefixHandler{prefix: a.prefix, next: a.r, internal: a.internal}
}

func (a *App) Router() *gin.Engine {
	return a.r
}

func (a *App) Run() {
	fmt.Printf("%s running at: http://127.0.0.1:%s%s\n", a.name, a.port, a.prefix)
	_ = http.ListenAndServe(":"+a.port, a.Handler())
}

// prefixHandler routes requests based on the configured sub-path prefix. This
// must happen at the http.Handler layer: Gin resolves a request to its route
// handler before any middleware runs, so the rewrite cannot be done inside a
// middleware.
//
// Prefixed requests are stripped of the prefix and delegated to the full router
// (public routes + health). Unprefixed requests go to the internal router, so
// only the health check answers at the bare root.
type prefixHandler struct {
	prefix   string
	next     http.Handler // Full router, reached after the prefix is stripped
	internal http.Handler // Internal-only router for unprefixed requests
}

func (h prefixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == h.prefix:
		path = "/"
	case strings.HasPrefix(path, h.prefix+"/"):
		path = strings.TrimPrefix(path, h.prefix)
	default:
		h.internal.ServeHTTP(w, r)
		return
	}

	r.URL.Path = path
	r.URL.RawPath = ""
	h.next.ServeHTTP(w, r)
}

// Default CORS middleware
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		MaxAge:           12 * time.Hour,
		AllowCredentials: true,
	})
}
