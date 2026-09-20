package ui_api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestShowcaseActionRendersShowcase(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/ui", ShowcaseAction)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ui", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected a text/html content type, got %q", ct)
	}

	body := w.Body.String()
	for _, want := range []string{
		"<title>airway-ui — component showcase</title>",
		`data-island="ui-showcase"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q", want)
		}
	}
}

func TestDemoItemsActionReturnsEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/api/v1/demo-items", DemoItemsAction)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/demo-items", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected a json content type, got %q", ct)
	}

	body := w.Body.String()
	for _, want := range []string{
		`"code":0`,
		`"items"`,
		`"SSD"`,
		`"Keyboard"`,
		`"Monitor"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %s", want, body)
		}
	}
}
