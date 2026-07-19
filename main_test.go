package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoadCLIEnvLoadsDotEnvWhenPresent(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	if err := os.Unsetenv("AIRWAY_DB_DSN"); err != nil {
		t.Fatalf("unset AIRWAY_DB_DSN: %v", err)
	}

	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(envPath, []byte("AIRWAY_DB_DSN=sqlite://./tmp/test.db\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	loadCLIEnv()

	if got := os.Getenv("AIRWAY_DB_DSN"); got != "sqlite://./tmp/test.db" {
		t.Fatalf("expected AIRWAY_DB_DSN from .env, got %q", got)
	}
}

func TestCORSAllowsViteFallbackPort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS())
	router.POST("/api/v1/admins", func(c *gin.Context) { c.Status(http.StatusCreated) })

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/admins", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5174")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)

	if response.Code != http.StatusNoContent {
		t.Fatalf("preflight: expected 204, got %d", response.Code)
	}
	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "http://127.0.0.1:5174" {
		t.Fatalf("unexpected allowed origin: %q", origin)
	}
}

func TestLoadCLIEnvDoesNotFailWhenDotEnvMissing(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(currentDir)
	})

	if err := os.Unsetenv("AIRWAY_DB_DSN"); err != nil {
		t.Fatalf("unset AIRWAY_DB_DSN: %v", err)
	}

	loadCLIEnv()

	if got := os.Getenv("AIRWAY_DB_DSN"); got != "" {
		t.Fatalf("expected AIRWAY_DB_DSN to remain empty, got %q", got)
	}
}
