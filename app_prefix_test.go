package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// When URL_PREFIX is configured, the public routes (home page, WebSocket, API)
// answer only under the prefix; the unprefixed root answers only the internal
// health check. The home page is served as HTML from a templ view, so these are
// substring-match smoke tests rather than full-body assertions.
func TestNewAppServesPublicRoutesUnderURLPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AIRWAY_URL_PREFIX", "")
	t.Setenv("URL_PREFIX", "/airway")

	app := NewApp("Airway", "0")
	r := app.Handler()

	okCases := []struct {
		name    string
		path    string
		wantSub string
	}{
		{"prefixed home with trailing slash", "/airway/", "Airway works!"},
		{"prefixed home without trailing slash", "/airway", "Airway works!"},
		{"prefixed health", "/airway/health", "UP"},
		{"unprefixed health stays reachable", "/health", "UP"},
	}

	for _, tc := range okCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("GET %s: expected 200, got %d: %s", tc.path, w.Code, w.Body.String())
			}
			if body := w.Body.String(); !strings.Contains(body, tc.wantSub) {
				t.Fatalf("GET %s: expected body to contain %q, got %q", tc.path, tc.wantSub, body)
			}
		})
	}

	// Public routes are prefix-only: the bare root and other unprefixed paths
	// must not serve the app.
	notFound := []struct {
		name   string
		method string
		path   string
	}{
		{"root home is not served", http.MethodGet, "/"},
		{"unprefixed websocket is not served", http.MethodGet, "/ws"},
		{"unprefixed API upload is not served", http.MethodPost, "/api/v1/storage"},
		{"unprefixed API download is not served", http.MethodGet, "/api/v1/storage/some/key"},
	}

	for _, tc := range notFound {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Fatalf("%s %s: expected 404, got %d: %s", tc.method, tc.path, w.Code, w.Body.String())
			}
		})
	}
}

func TestNewAppWithoutPrefixServesEverythingAtRoot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AIRWAY_URL_PREFIX", "")
	t.Setenv("URL_PREFIX", "")

	app := NewApp("Airway", "0")
	r := app.Handler()

	for _, tc := range []struct {
		path    string
		wantSub string
	}{
		{"/", "Airway works!"},
		{"/health", "UP"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d: %s", tc.path, w.Code, w.Body.String())
		}
		if body := w.Body.String(); !strings.Contains(body, tc.wantSub) {
			t.Fatalf("GET %s: expected body to contain %q, got %q", tc.path, tc.wantSub, body)
		}
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
