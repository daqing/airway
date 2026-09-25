package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daqing/airway/config"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func newTestApp() *App {
	return NewApp("Airway", WithRoutes(config.Routes, config.HealthRoutes))
}

func TestValidateListenAddress(t *testing.T) {
	for _, tc := range []struct {
		listen string
		ok     bool
	}{
		{":1900", true},
		{"0.0.0.0:1905", true},
		{"127.0.0.1:1905", true},
		{"[::1]:1905", true},
		{"1905", false},
		{"0.0.0.0", false},
		{"::1905", false},
	} {
		err := validateListenAddress(tc.listen)
		if tc.ok && err != nil {
			t.Fatalf("validateListenAddress(%q) = %v, want nil", tc.listen, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("validateListenAddress(%q) = nil, want an error", tc.listen)
		}
	}
}

func TestBrowsableAddr(t *testing.T) {
	for _, tc := range []struct {
		listen string
		want   string
	}{
		{":1900", "127.0.0.1:1900"},
		{"0.0.0.0:1905", "127.0.0.1:1905"},
		{"[::]:1905", "127.0.0.1:1905"},
		{"192.168.1.5:1905", "192.168.1.5:1905"},
		{"example.com:1905", "example.com:1905"},
		{"not-an-address", "not-an-address"},
	} {
		if got := browsableAddr(tc.listen); got != tc.want {
			t.Fatalf("browsableAddr(%q) = %q, want %q", tc.listen, got, tc.want)
		}
	}
}

func TestRunBanner(t *testing.T) {
	for _, tc := range []struct {
		listen string
		want   string
	}{
		// The configured address is echoed verbatim; wildcard binds get the
		// loopback URL appended as a hint.
		{"0.0.0.0:1999", "0.0.0.0:1999 (http://127.0.0.1:1999)"},
		{":1900", ":1900 (http://127.0.0.1:1900)"},
		{"[::]:1905", "[::]:1905 (http://127.0.0.1:1905)"},
		{"127.0.0.1:1905", "http://127.0.0.1:1905"},
		{"192.168.1.5:1905", "http://192.168.1.5:1905"},
	} {
		if got := runBanner(tc.listen, ""); got != tc.want {
			t.Fatalf("runBanner(%q, \"\") = %q, want %q", tc.listen, got, tc.want)
		}
	}

	if got, want := runBanner("0.0.0.0:1905", "/airway"), "0.0.0.0:1905 (http://127.0.0.1:1905/airway)"; got != want {
		t.Fatalf("runBanner with prefix = %q, want %q", got, want)
	}
}

func TestNewAppUsesResolvedListenAddress(t *testing.T) {
	t.Setenv("LISTEN", "0.0.0.0:1905")

	// The App stores whatever the resolver returns, so the test stays honest
	// even when the outer environment shadows the value.
	if got, want := newTestApp().listen, utils.ListenAddress(); got != want {
		t.Fatalf("listen = %q, want %q", got, want)
	}
}

// When URL_PREFIX is configured, the public routes (home page, WebSocket, API)
// answer only under the prefix; the unprefixed root answers only the internal
// health check. The home page is served as HTML from a templ view, so these are
// substring-match smoke tests rather than full-body assertions.
func TestNewAppServesPublicRoutesUnderURLPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AIRWAY_URL_PREFIX", "")
	t.Setenv("URL_PREFIX", "/airway")

	app := newTestApp()
	r := app.Handler()

	okCases := []struct {
		name    string
		path    string
		wantSub string
	}{
		{"prefixed home with trailing slash", "/airway/", "Less setup."},
		{"prefixed home without trailing slash", "/airway", "Less setup."},
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

	app := newTestApp()
	r := app.Handler()

	for _, tc := range []struct {
		path    string
		wantSub string
	}{
		{"/", "Less setup."},
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

	app := newTestApp()
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

func TestStrictOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// The middleware aborts cross-origin requests and passes same-origin ones.
	router := gin.New()
	router.Use(StrictOrigin())
	router.POST("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "http://evil.example")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin POST: expected 403, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	req.Host = "127.0.0.1:5100"
	req.Header.Set("Origin", "http://127.0.0.1:5100")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("same-origin POST: expected 200, got %d", w.Code)
	}
}
