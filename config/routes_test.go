package config

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRoutesRegistersCoreEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	Routes(r)

	registered := map[string]bool{}
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"GET /",
		"GET /health",
		"GET /openapi.json",
		"GET /ws",
		"POST /ws/publish",
		"POST /api/v1/storage",
		"GET /api/v1/storage/*key",
		"DELETE /api/v1/storage/*key",
	}

	for _, route := range expected {
		if !registered[route] {
			t.Fatalf("expected route %s to be registered, got %#v", route, registered)
		}
	}
}

func TestStaticRoutesUseConfiguredPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("STATIC_URL_PREFIX", "/downloads/")

	r := gin.New()
	Routes(r)

	for _, route := range r.Routes() {
		if route.Method == "GET" && route.Path == "/downloads/*path" {
			return
		}
	}
	t.Fatalf("expected GET /downloads/*path to be registered, got %#v", r.Routes())
}
