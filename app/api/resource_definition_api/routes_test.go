package resource_definition_api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daqing/airway/app/api/resource_definition_api"
	"github.com/daqing/airway/app/middleware"
	"github.com/daqing/airway/app/modules/dynamicresource"
	"github.com/daqing/airway/app/modules/identity"
	_ "github.com/daqing/airway/db/migrate"
	"github.com/daqing/airway/lib/migrate/dialect"
	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func setupRouter(t *testing.T) (*gin.Engine, string, string) {
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
				t.Fatal(err)
			}
			for _, statement := range statements {
				if _, err := db.Conn().Exec(statement); err != nil {
					t.Fatalf("migration %q: %v", statement, err)
				}
			}
		}
	}
	identityService := identity.NewService(db, 12*time.Hour)
	super, err := identityService.Initialize(context.Background(), "root", "root@example.com", "very-secure-password", "test", "")
	if err != nil {
		t.Fatal(err)
	}
	_, superToken, _, err := identityService.Login(context.Background(), super.Login, "very-secure-password", "test", "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	digest, err := utils.EncryptPassword("very-secure-password")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Conn().Exec(`INSERT INTO admins (login,email,password_digest,status,super_admin,auth_version,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`, "member", "member@example.com", digest, "active", false, 1, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, memberToken, _, err := identityService.Login(context.Background(), "member", "very-secure-password", "test", "")
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1")
	group.Use(middleware.Authenticate(identityService))
	resource_definition_api.Routes(group, dynamicresource.NewService(db))
	return router, superToken, memberToken
}

func request(router *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "airway_session", Value: token})
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func validDefinition() map[string]any {
	return map[string]any{"code": "articles", "name": "文章", "table_name": "articles", "schema": map[string]any{"fields": []map[string]any{{"code": "title", "name": "标题", "type": "string", "required": true, "list": true, "searchable": true, "filterable": true, "sortable": true, "input": "text"}, {"code": "published", "name": "已发布", "type": "boolean", "required": true, "list": true, "filterable": true}}, "permissions": map[string]any{"list": "articles:list", "read": "articles:read", "create": "articles:create", "update": "articles:update", "delete": "articles:delete"}}}
}

func TestResourceDefinitionLifecycle(t *testing.T) {
	router, superToken, memberToken := setupRouter(t)
	if response := request(router, http.MethodGet, "/api/v1/resource-definitions", nil, ""); response.Code != 401 {
		t.Fatalf("anonymous expected 401, got %d", response.Code)
	}
	if response := request(router, http.MethodGet, "/api/v1/resource-definitions", nil, memberToken); response.Code != 403 {
		t.Fatalf("regular administrator expected 403, got %d", response.Code)
	}
	created := request(router, http.MethodPost, "/api/v1/resource-definitions", validDefinition(), superToken)
	if created.Code != 201 {
		t.Fatalf("create expected 201, got %d: %s", created.Code, created.Body.String())
	}
	var payload struct {
		Data dynamicresource.Definition `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	item := payload.Data
	if item.Status != "draft" {
		t.Fatalf("expected draft, got %s", item.Status)
	}
	validated := request(router, http.MethodPost, fmt.Sprintf("/api/v1/resource-definitions/%d/validate", item.ID), nil, superToken)
	if validated.Code != 200 {
		t.Fatalf("validate expected 200, got %d: %s", validated.Code, validated.Body.String())
	}
	published := request(router, http.MethodPost, fmt.Sprintf("/api/v1/resource-definitions/%d/publish", item.ID), nil, superToken)
	if published.Code != 200 {
		t.Fatalf("publish expected 200, got %d: %s", published.Code, published.Body.String())
	}
	if err := json.Unmarshal(published.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Status != "published" || payload.Data.ActiveVersion == nil || *payload.Data.ActiveVersion != 1 {
		t.Fatalf("unexpected published definition: %#v", payload.Data)
	}
	update := request(router, http.MethodPatch, fmt.Sprintf("/api/v1/resource-definitions/%d", item.ID), validDefinition(), superToken)
	if update.Code != 409 {
		t.Fatalf("published update expected 409, got %d", update.Code)
	}
	versions := request(router, http.MethodGet, fmt.Sprintf("/api/v1/resource-definitions/%d/versions", item.ID), nil, superToken)
	if versions.Code != 200 {
		t.Fatalf("versions expected 200, got %d", versions.Code)
	}
	var versionsPayload struct {
		Data []dynamicresource.Version `json:"data"`
	}
	_ = json.Unmarshal(versions.Body.Bytes(), &versionsPayload)
	if len(versionsPayload.Data) != 1 || versionsPayload.Data[0].Checksum == "" {
		t.Fatalf("unexpected versions: %#v", versionsPayload.Data)
	}
	deactivated := request(router, http.MethodPost, fmt.Sprintf("/api/v1/resource-definitions/%d/deactivate", item.ID), nil, superToken)
	if deactivated.Code != 200 {
		t.Fatalf("deactivate expected 200, got %d: %s", deactivated.Code, deactivated.Body.String())
	}
	_ = json.Unmarshal(deactivated.Body.Bytes(), &payload)
	if payload.Data.Status != "inactive" {
		t.Fatalf("expected inactive, got %s", payload.Data.Status)
	}
}

func TestValidationErrorsPreventPublish(t *testing.T) {
	router, superToken, _ := setupRouter(t)
	invalid := validDefinition()
	invalid["table_name"] = "unsafe-table"
	schemaBody := invalid["schema"].(map[string]any)
	schemaBody["fields"] = []map[string]any{{"code": "id", "name": "ID", "type": "unknown"}}
	created := request(router, http.MethodPost, "/api/v1/resource-definitions", invalid, superToken)
	var payload struct {
		Data dynamicresource.Definition `json:"data"`
	}
	_ = json.Unmarshal(created.Body.Bytes(), &payload)
	validated := request(router, http.MethodPost, fmt.Sprintf("/api/v1/resource-definitions/%d/validate", payload.Data.ID), nil, superToken)
	if validated.Code != 422 {
		t.Fatalf("invalid validate expected 422, got %d: %s", validated.Code, validated.Body.String())
	}
	published := request(router, http.MethodPost, fmt.Sprintf("/api/v1/resource-definitions/%d/publish", payload.Data.ID), nil, superToken)
	if published.Code != 409 {
		t.Fatalf("unvalidated publish expected 409, got %d", published.Code)
	}
}
