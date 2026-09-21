package boot

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func desktopTestOptions(t *testing.T, out io.Writer) Options {
	t.Helper()
	t.Setenv("AIRWAY_ENV", "")
	t.Setenv("STORAGE_DRIVER", "")
	t.Setenv("STORAGE_ROOT", "")

	root := t.TempDir()

	return Options{
		AppName: "BootTest",
		Env:     "production",
		DSN:     "sqlite://" + filepath.ToSlash(filepath.Join(root, "app.db")),
		Migrations: fstest.MapFS{
			"0001_create_notes.up.sql":   {Data: []byte("CREATE TABLE notes (id INTEGER PRIMARY KEY, title VARCHAR(255) NOT NULL);")},
			"0001_create_notes.down.sql": {Data: []byte("DROP TABLE notes;")},
		},
		StorageRoot:  filepath.Join(root, "storage"),
		Out:          out,
		Routes: func(r *gin.Engine) {
			r.GET("/boot-ok", func(c *gin.Context) { c.String(http.StatusOK, "boot-ok") })
			r.GET("/boot-html", func(c *gin.Context) {
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.String(http.StatusOK, "<html><head></head><body>hi</body></html>")
			})
			r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "UP") })
		},
		HealthRoutes: func(r *gin.Engine) { r.GET("/health", func(c *gin.Context) { c.String(http.StatusOK, "UP") }) },
	}
}

func TestNewBootsDesktopStack(t *testing.T) {
	a, err := New(desktopTestOptions(t, io.Discard))
	if err != nil {
		t.Fatalf("boot: %v", err)
	}

	r := a.Handler()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "UP") {
		t.Fatalf("GET /health: expected 200 UP, got %d: %s", w.Code, w.Body.String())
	}

	// The project's own routes are mounted.
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boot-ok", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /boot-ok: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNewStrictCORSRejectsForeignOrigins(t *testing.T) {
	opts := desktopTestOptions(t, io.Discard)
	opts.StrictCORS = true
	opts.SameOriginWS = true

	a, err := New(opts)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage", nil)
	req.Header.Set("Origin", "http://evil.example")
	a.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin POST: expected 403, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/storage", nil)
	req.Host = "127.0.0.1:1900"
	req.Header.Set("Origin", "http://127.0.0.1:1900")
	a.Handler().ServeHTTP(w, req)
	if w.Code == http.StatusForbidden {
		t.Fatalf("same-origin POST: unexpectedly rejected with 403")
	}
}

func TestNewRequiresEnv(t *testing.T) {
	t.Setenv("AIRWAY_ENV", "")

	opts := desktopTestOptions(t, io.Discard)
	opts.Env = ""

	if _, err := New(opts); err == nil || !strings.Contains(err.Error(), "AIRWAY_ENV is not set") {
		t.Fatalf("expected missing-env error, got: %v", err)
	}
}

func TestNewFailsOnBrokenMigrations(t *testing.T) {
	opts := desktopTestOptions(t, io.Discard)
	opts.Migrations = fstest.MapFS{
		"0001_broken.up.sql":   {Data: []byte("CREATE TABLE notes (")},
		"0001_broken.down.sql": {Data: []byte("DROP TABLE notes;")},
	}

	if _, err := New(opts); err == nil || !strings.Contains(err.Error(), "migrate") {
		t.Fatalf("expected migration error, got: %v", err)
	}
}

func TestNewHideScrollbarsInjectsCSSIntoHTMLOnly(t *testing.T) {
	opts := desktopTestOptions(t, io.Discard)
	opts.HideScrollbars = true

	a, err := New(opts)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boot-html", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /boot-html: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "::-webkit-scrollbar") {
		t.Fatalf("expected scrollbar CSS injected into the HTML page, got:\n%s", w.Body.String())
	}

	// Non-HTML responses pass through untouched.
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boot-ok", nil))
	if strings.Contains(w.Body.String(), "::-webkit-scrollbar") {
		t.Fatalf("expected plain-text responses to be untouched, got:\n%s", w.Body.String())
	}
}

// The desktop binary must mount the host project's routes, not the
// framework's built-in demo pages.
func TestNewMountsProjectRoutes(t *testing.T) {
	opts := desktopTestOptions(t, io.Discard)
	opts.Routes = func(r *gin.Engine) {
		r.GET("/project-home", func(c *gin.Context) { c.String(http.StatusOK, "project marker") })
	}

	a, err := New(opts)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}

	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/project-home", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "project marker") {
		t.Fatalf("GET /project-home: expected the project route to be mounted, got %d: %s", w.Code, w.Body.String())
	}

	// The framework's demo landing page must NOT be mounted.
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if strings.Contains(w.Body.String(), "Less setup.") {
		t.Fatalf("expected the framework demo page to be absent, got:\n%s", w.Body.String())
	}
}

func TestNewKeepsCurrentEnvWhenUnpinned(t *testing.T) {
	opts := desktopTestOptions(t, io.Discard)
	opts.Env = ""
	t.Setenv("AIRWAY_ENV", "production")

	if _, err := New(opts); err != nil {
		t.Fatalf("boot: %v", err)
	}

	if os.Getenv("AIRWAY_ENV") != "production" {
		t.Fatalf("expected AIRWAY_ENV to stay untouched, got %q", os.Getenv("AIRWAY_ENV"))
	}
}
