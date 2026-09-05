package home_api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIndexActionRendersHomePage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/", IndexAction)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected a text/html content type, got %q", ct)
	}

	body := w.Body.String()
	for _, want := range []string{
		"<title>Airway — The full-stack Go web framework</title>",
		"Less setup.",
		"THE FULL-STACK GO WEB FRAMEWORK",
		"id=\"features\"",
		"id=\"get-started\"",
		"PostgreSQL",
		"MySQL 8",
		"SQLite",
		"Cloudflare R2",
		"WebSocket",
		"REPL",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}
