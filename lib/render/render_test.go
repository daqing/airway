package render

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type testResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestOKRendersZeroCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	OK(c, gin.H{"name": "airway"})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp testResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Code != 0 || resp.Message != "" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestErrorMessageRendersGeneralErrorCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	ErrorMessage(c, "boom")

	var resp testResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.Code != ErrGeneral || resp.Message != "boom" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
