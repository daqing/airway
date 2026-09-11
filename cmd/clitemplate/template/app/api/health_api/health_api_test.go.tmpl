package health_api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthActionReturnsUP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	Routes(r)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if body := w.Body.String(); body != "UP\n" {
		t.Fatalf("expected body %q, got %q", "UP\n", body)
	}
}
