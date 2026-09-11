package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPublishRejectsEmptyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/ws/publish", nil)

	Publish(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestPublishAcceptsMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/ws/publish", strings.NewReader("message=hello"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request = req

	Publish(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if body := w.Body.String(); !strings.Contains(body, "Message sent") {
		t.Fatalf("expected body to contain %q, got %q", "Message sent", body)
	}
}
