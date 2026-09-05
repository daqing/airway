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
	r      *gin.Engine
	name   string // Application name
	port   string
	prefix string // Public sub-path prefix ("" = serve at root)
}

func NewApp(name, port string) *App {
	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(CORS())

	config.Routes(router)

	return &App{
		r:      router,
		name:   name,
		port:   port,
		prefix: utils.URLPrefix(),
	}
}

// Handler returns the http.Handler that serves the app. When a public sub-path
// prefix is configured (e.g. "/airway") requests carrying that prefix are
// stripped before they reach Gin, so routes stay registered at the root while
// the app answers under the prefix. Requests without the prefix (such as
// internal health checks) pass through unchanged.
func (a *App) Handler() http.Handler {
	if a.prefix == "" {
		return a.r
	}

	return prefixHandler{prefix: a.prefix, next: a.r}
}

func (a *App) Router() *gin.Engine {
	return a.r
}

func (a *App) Run() {
	fmt.Printf("%s running at: http://127.0.0.1:%s%s\n", a.name, a.port, a.prefix)
	_ = http.ListenAndServe(":"+a.port, a.Handler())
}

// prefixHandler strips the configured sub-path prefix from an incoming request
// before delegating to the underlying Gin engine. This must happen at the
// http.Handler layer: Gin resolves a request to its route handler before any
// middleware runs, so the rewrite cannot be done inside a middleware.
type prefixHandler struct {
	prefix string
	next   http.Handler
}

func (h prefixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == h.prefix:
		path = "/"
	case strings.HasPrefix(path, h.prefix+"/"):
		path = strings.TrimPrefix(path, h.prefix)
	default:
		h.next.ServeHTTP(w, r)
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
