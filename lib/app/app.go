// Package app wires the Airway HTTP stack: engine construction, middleware,
// routes and the public http.Handler. It is a library package so the web
// binary and desktop wrappers share one implementation, and it deliberately
// imports no application package: the route source is always supplied by the
// caller, so each binary compiles exactly one set of API modules.
package app

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/daqing/airway/lib/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type App struct {
	r        *gin.Engine                       // Full router: public + internal routes at the root
	internal *gin.Engine                       // Internal-only router (health) for unprefixed requests
	name     string                            // Application name
	listen   string                            // Listen address (host:port), from LISTEN
	prefix   string                            // Public sub-path prefix ("" = serve at root)
	outer    []func(http.Handler) http.Handler // Wrappers around the outermost handler
}

type Option func(*options)

type options struct {
	cors         gin.HandlerFunc
	routes       func(*gin.Engine)
	healthRoutes func(*gin.Engine)
	outer        []func(http.Handler) http.Handler
}

// WithCORS replaces the default permissive CORS middleware. Desktop wrappers
// pass StrictOrigin; anything unconfigured keeps the web default.
func WithCORS(h gin.HandlerFunc) Option {
	return func(o *options) { o.cors = h }
}

// WithRoutes sets the route source. It is REQUIRED: the framework repo's web
// binary passes its own config.Routes, and desktop wrappers pass the host
// project's config.Routes, so every binary mounts exactly one set of API
// modules (their init()s also register OpenAPI declarations — mounting two
// sets would double-declare the same routes). healthRoutes may be nil when
// no internal health check is wanted.
func WithRoutes(routes, healthRoutes func(*gin.Engine)) Option {
	return func(o *options) {
		o.routes = routes
		o.healthRoutes = healthRoutes
	}
}

// WithHandlerWrapper appends a wrapper applied around the app's outermost
// http.Handler (prefix handling included). Desktop wrappers use it for
// response post-processing such as CSS injection.
func WithHandlerWrapper(wrap func(http.Handler) http.Handler) Option {
	return func(o *options) { o.outer = append(o.outer, wrap) }
}

func NewApp(name string, opts ...Option) *App {
	cfg := options{}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.routes == nil {
		panic("app: no routes configured — pass app.WithRoutes(config.Routes, config.HealthRoutes)")
	}

	listen := utils.ListenAddress()
	if err := validateListenAddress(listen); err != nil {
		panic("app: " + err.Error())
	}

	router := newEngine(cfg.cors)
	cfg.routes(router)

	internal := newEngine(cfg.cors)
	if cfg.healthRoutes != nil {
		cfg.healthRoutes(internal)
	}

	return &App{
		r:        router,
		internal: internal,
		name:     name,
		listen:   listen,
		prefix:   utils.URLPrefix(),
		outer:    cfg.outer,
	}
}

func newEngine(corsMiddleware gin.HandlerFunc) *gin.Engine {
	if corsMiddleware == nil {
		corsMiddleware = CORS()
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware)
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
	var h http.Handler
	if a.prefix == "" {
		h = a.r
	} else {
		h = prefixHandler{prefix: a.prefix, next: a.r, internal: a.internal}
	}

	for i := len(a.outer) - 1; i >= 0; i-- {
		h = a.outer[i](h)
	}

	return h
}

func (a *App) Router() *gin.Engine {
	return a.r
}

func (a *App) Run() {
	fmt.Printf("%s running at: %s\n", a.name, runBanner(a.listen, a.prefix))

	if err := http.ListenAndServe(a.listen, a.Handler()); err != nil {
		log.Fatalf("%s: listen on %s failed: %v", a.name, a.listen, err)
	}
}

// runBanner renders the address part of the startup log line: the configured
// LISTEN address verbatim, so the log always echoes what the server bound.
// Wildcard binds (":1999", "0.0.0.0:1999") additionally get the loopback URL,
// which is the one a browser on this machine can actually open.
func runBanner(listen, prefix string) string {
	url := "http://" + browsableAddr(listen) + prefix

	if host, _, ok := utils.ListenHostPort(listen); ok && host == "" {
		return fmt.Sprintf("%s (%s)", listen, url)
	}

	return url
}

// browsableAddr turns a listen address into the URL a developer should open:
// wildcard hosts ("", "0.0.0.0", "::") become 127.0.0.1, so a
// LISTEN=0.0.0.0:1905 log line still points somewhere clickable.
func browsableAddr(listen string) string {
	host, port, ok := utils.ListenHostPort(listen)
	if !ok {
		return listen
	}

	if host == "" {
		host = "127.0.0.1"
	}

	return net.JoinHostPort(host, port)
}

// validateListenAddress checks the address the server will bind: LISTEN
// carries host and port together, so a bare port ("1905") is rejected.
func validateListenAddress(listen string) error {
	if _, _, err := net.SplitHostPort(listen); err != nil {
		return fmt.Errorf("invalid listen address %q — use host:port, e.g. 0.0.0.0:1905 or :1905", listen)
	}

	return nil
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

// StrictOrigin rejects requests whose Origin does not match the request Host.
// Desktop windows load the app from its own 127.0.0.1 origin, so everything
// they send is same-origin; cross-origin calls can only come from other local
// pages and have no business reaching the embedded server.
func StrictOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if utils.SameOriginRequest(c.Request) {
			c.Next()
			return
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}
