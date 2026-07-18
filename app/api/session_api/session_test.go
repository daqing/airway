package session_api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daqing/airway/app/api/session_api"
	"github.com/daqing/airway/app/modules/identity"
	_ "github.com/daqing/airway/db/migrate"
	"github.com/daqing/airway/lib/migrate/dialect"
	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

// setupRouter creates an isolated test environment with identity tables and session routes.
func setupRouter(t *testing.T) (*gin.Engine, *repo.DB) {
	t.Helper()
	db, err := repo.SetupDB("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	compiler := dialect.NewCompiler(db.Driver())
	for _, definition := range schema.Definitions() {
		for _, op := range definition.UpOps {
			statements, err := compiler.Compile(op)
			if err != nil {
				t.Fatalf("compile migration: %v", err)
			}
			for _, statement := range statements {
				if _, err := db.Conn().Exec(statement); err != nil {
					t.Fatalf("execute migration %q: %v", statement, err)
				}
			}
		}
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	session_api.Routes(router.Group("/api/v1"))
	return router, db
}

// request sends a JSON request to the test router and returns the response recorder.
func request(t *testing.T, router *gin.Engine, method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-request")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

// TestInitializationSessionAndAuditFlow verifies the full session and audit flow after CLI setup.
func TestInitializationSessionAndAuditFlow(t *testing.T) {
	router, db := setupRouter(t)
	service := identity.NewService(db, 12*time.Hour)
	initialized, err := service.IsInitialized(context.Background())
	if err != nil || initialized {
		t.Fatalf("initial state: initialized=%v err=%v", initialized, err)
	}
	if _, err := service.Initialize(context.Background(), "root", "root@example.com", "very-secure-password", "cli", ""); err != nil {
		t.Fatalf("initialize from CLI service: %v", err)
	}
	initialized, err = service.IsInitialized(context.Background())
	if err != nil || !initialized {
		t.Fatalf("initialized state: initialized=%v err=%v", initialized, err)
	}

	response := request(t, router, http.MethodPost, "/api/v1/setup/super-admin", map[string]string{}, nil)
	if response.Code != http.StatusNotFound {
		t.Fatalf("super admin must not be initialized over HTTP, got %d", response.Code)
	}

	response = request(t, router, http.MethodPost, "/api/v1/sessions", map[string]string{"login": "root", "password": "very-secure-password"}, nil)
	if response.Code != http.StatusCreated {
		t.Fatalf("login: got %d: %s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly {
		t.Fatalf("expected HttpOnly session cookie, got %#v", cookies)
	}

	response = request(t, router, http.MethodGet, "/api/v1/session", nil, cookies[0])
	if response.Code != http.StatusOK {
		t.Fatalf("current session: got %d: %s", response.Code, response.Body.String())
	}
	response = request(t, router, http.MethodDelete, "/api/v1/session", nil, cookies[0])
	if response.Code != http.StatusNoContent {
		t.Fatalf("logout: got %d: %s", response.Code, response.Body.String())
	}
	response = request(t, router, http.MethodGet, "/api/v1/session", nil, cookies[0])
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session: got %d", response.Code)
	}

	var auditCount int
	if err := db.Conn().Get(&auditCount, `SELECT COUNT(*) FROM audit_logs WHERE result='success'`); err != nil || auditCount != 3 {
		t.Fatalf("successful audit events: count=%d err=%v", auditCount, err)
	}
}

// TestLoginRateLimit verifies that repeated login failures trigger rate limiting and auditing.
func TestLoginRateLimit(t *testing.T) {
	router, db := setupRouter(t)
	for i := 0; i < 5; i++ {
		response := request(t, router, http.MethodPost, "/api/v1/sessions", map[string]string{"login": "attacker", "password": "wrong"}, nil)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: got %d", i+1, response.Code)
		}
	}
	response := request(t, router, http.MethodPost, "/api/v1/sessions", map[string]string{"login": "attacker", "password": "wrong"}, nil)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("limited attempt: got %d: %s", response.Code, response.Body.String())
	}
	var count int
	if err := db.Conn().Get(&count, `SELECT COUNT(*) FROM audit_logs WHERE action='auth.login_rate_limited'`); err != nil || count != 1 {
		t.Fatalf("rate limit audit: count=%d err=%v", count, err)
	}
}
