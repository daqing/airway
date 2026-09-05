package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewAppServesUnderURLPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AIRWAY_URL_PREFIX", "")
	t.Setenv("URL_PREFIX", "/airway")

	app := NewApp("Airway", "0")
	r := app.Handler()

	cases := []struct {
		name string
		path string
		want string
	}{
		{"prefixed home with trailing slash", "/airway/", "Hello, Airway!"},
		{"prefixed home without trailing slash", "/airway", "Hello, Airway!"},
		{"prefixed health", "/airway/health", "UP\n"},
		{"root home still reachable", "/", "Hello, Airway!"},
		{"root health still reachable", "/health", "UP\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("GET %s: expected 200, got %d: %s", tc.path, w.Code, w.Body.String())
			}
			if body := w.Body.String(); body != tc.want {
				t.Fatalf("GET %s: expected body %q, got %q", tc.path, tc.want, body)
			}
		})
	}
}

func TestNewAppRegistersRoutesUnderPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("URL_PREFIX", "/airway")

	app := NewApp("Airway", "0")
	r := app.Handler()

	// POST /ws/publish with no payload reaches the handler (a 4xx proves the
	// route matched after the prefix was stripped rather than a 404).
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/airway/ws/publish", nil)
	r.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Fatalf("POST /airway/ws/publish: expected the route to be matched, got 404")
	}
}
