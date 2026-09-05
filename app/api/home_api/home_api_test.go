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
	// The h1 carries a templ-generated class, so match the text and closing tag
	// rather than a bare <h1> tag. height:100vh + flex is the full-viewport
	// centering rule from the indexPage CSS class.
	for _, want := range []string{"<title>Airway</title>", ">Airway works!</h1>", "height:100vh", "display:flex"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}
