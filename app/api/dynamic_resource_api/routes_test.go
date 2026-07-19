package dynamic_resource_api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daqing/airway/app/api/dynamic_resource_api"
	"github.com/daqing/airway/app/middleware"
	"github.com/daqing/airway/app/modules/dynamicresource"
	"github.com/daqing/airway/app/modules/identity"
	"github.com/daqing/airway/app/modules/rbac"
	_ "github.com/daqing/airway/db/migrate"
	"github.com/daqing/airway/lib/migrate/dialect"
	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

func setup(t *testing.T) (*gin.Engine, *repo.DB, string) {
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
	identities := identity.NewService(db, 12*time.Hour)
	admin, err := identities.Initialize(context.Background(), "root", "root@example.com", "very-secure-password", "test", "")
	if err != nil {
		t.Fatal(err)
	}
	_, token, _, err := identities.Login(context.Background(), admin.Login, "very-secure-password", "test", "")
	if err != nil {
		t.Fatal(err)
	}
	resources := dynamicresource.NewService(db)
	for _, example := range dynamicresource.Examples() {
		definition, err := resources.Create(context.Background(), example.Code, example.Name, example.TableName, example.Schema)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = resources.Validate(context.Background(), definition.ID); err != nil {
			t.Fatal(err)
		}
		if _, err = resources.Publish(context.Background(), definition.ID, admin.ID); err != nil {
			t.Fatal(err)
		}
	}
	// Simulate a definition published by an older release which did not create
	// the physical table. A fresh runtime service must repair it on first use.
	if _, err := db.Conn().Exec(`DROP TABLE articles`); err != nil {
		t.Fatal(err)
	}
	runtimeResources := dynamicresource.NewService(db)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1")
	group.Use(middleware.Authenticate(identities))
	dynamic_resource_api.Routes(group, runtimeResources, rbac.NewService(db))
	return router, db, token
}
func do(router *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
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

func TestGeneratedCRUDAPI(t *testing.T) {
	router, _, token := setup(t)
	created := do(router, http.MethodPost, "/api/v1/resources/articles/records", map[string]any{"title": "First article", "body": "Article body", "views": 10, "published": false}, token)
	if created.Code != 201 {
		t.Fatalf("create expected 201, got %d: %s", created.Code, created.Body.String())
	}
	var payload struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	id := int64(payload.Data["id"].(float64))
	if payload.Data["title"] != "First article" {
		t.Fatalf("unexpected created record: %#v", payload.Data)
	}
	listed := do(router, http.MethodGet, "/api/v1/resources/articles/records", nil, token)
	if listed.Code != 200 {
		t.Fatalf("list expected 200, got %d: %s", listed.Code, listed.Body.String())
	}
	var listPayload struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(listed.Body.Bytes(), &listPayload)
	if len(listPayload.Data) != 1 {
		t.Fatalf("expected one record, got %#v", listPayload.Data)
	}
	detail := do(router, http.MethodGet, fmt.Sprintf("/api/v1/resources/articles/records/%d", id), nil, token)
	if detail.Code != 200 {
		t.Fatalf("detail expected 200, got %d", detail.Code)
	}
	updated := do(router, http.MethodPatch, fmt.Sprintf("/api/v1/resources/articles/records/%d", id), map[string]any{"title": "Updated article", "lock_version": payload.Data["lock_version"]}, token)
	if updated.Code != 200 {
		t.Fatalf("update expected 200, got %d: %s", updated.Code, updated.Body.String())
	}
	_ = json.Unmarshal(updated.Body.Bytes(), &payload)
	if payload.Data["title"] != "Updated article" || payload.Data["lock_version"].(float64) != 2 {
		t.Fatalf("unexpected update: %#v", payload.Data)
	}
	deleted := do(router, http.MethodDelete, fmt.Sprintf("/api/v1/resources/articles/records/%d", id), nil, token)
	if deleted.Code != 204 {
		t.Fatalf("delete expected 204, got %d: %s", deleted.Code, deleted.Body.String())
	}
	missing := do(router, http.MethodGet, fmt.Sprintf("/api/v1/resources/articles/records/%d", id), nil, token)
	if missing.Code != 404 {
		t.Fatalf("deleted detail expected 404, got %d", missing.Code)
	}
}

func TestFieldWhitelistPaginationConflictAndAudit(t *testing.T) {
	router, db, token := setup(t)

	unknown := do(router, http.MethodPost, "/api/v1/resources/articles/records", map[string]any{"title": "Invalid", "body": "Body", "published": false, "unknown": "value"}, token)
	if unknown.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown field expected 422, got %d: %s", unknown.Code, unknown.Body.String())
	}

	now := time.Now().UTC()
	for i := 0; i < 105; i++ {
		if _, err := db.Conn().Exec(`INSERT INTO articles (title,body,views,published,lock_version,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`, fmt.Sprintf("Article %d", i), "Body", i, false, 1, now, now); err != nil {
			t.Fatal(err)
		}
	}
	pageOne := do(router, http.MethodGet, "/api/v1/resources/articles/records?page_size=1000", nil, token)
	if pageOne.Code != 200 {
		t.Fatalf("page one: %d %s", pageOne.Code, pageOne.Body.String())
	}
	var pagePayload struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			Page     int `json:"page"`
			PageSize int `json:"page_size"`
			Total    int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(pageOne.Body.Bytes(), &pagePayload); err != nil {
		t.Fatal(err)
	}
	if len(pagePayload.Data) != 100 || pagePayload.Meta.PageSize != 100 || pagePayload.Meta.Total != 105 {
		t.Fatalf("unexpected capped page: len=%d meta=%#v", len(pagePayload.Data), pagePayload.Meta)
	}
	pageTwo := do(router, http.MethodGet, "/api/v1/resources/articles/records?page=2&page_size=100", nil, token)
	_ = json.Unmarshal(pageTwo.Body.Bytes(), &pagePayload)
	if len(pagePayload.Data) != 5 {
		t.Fatalf("expected 5 records on second page, got %d", len(pagePayload.Data))
	}

	created := do(router, http.MethodPost, "/api/v1/resources/articles/records", map[string]any{"title": "Concurrent", "body": "Body", "views": 1, "published": false}, token)
	var recordPayload struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(created.Body.Bytes(), &recordPayload)
	id := int64(recordPayload.Data["id"].(float64))
	version := recordPayload.Data["lock_version"]
	first := do(router, http.MethodPatch, fmt.Sprintf("/api/v1/resources/articles/records/%d", id), map[string]any{"title": "First update", "lock_version": version}, token)
	if first.Code != 200 {
		t.Fatalf("first update: %d %s", first.Code, first.Body.String())
	}
	stale := do(router, http.MethodPatch, fmt.Sprintf("/api/v1/resources/articles/records/%d", id), map[string]any{"title": "Stale update", "lock_version": version}, token)
	if stale.Code != 409 {
		t.Fatalf("stale update expected 409, got %d: %s", stale.Code, stale.Body.String())
	}
	deleted := do(router, http.MethodDelete, fmt.Sprintf("/api/v1/resources/articles/records/%d", id), nil, token)
	if deleted.Code != 204 {
		t.Fatalf("delete: %d", deleted.Code)
	}

	var actions []string
	if err := db.Conn().Select(&actions, `SELECT action FROM audit_logs WHERE target_type='articles' ORDER BY id`); err != nil {
		t.Fatal(err)
	}
	want := []string{"resource.record_create", "resource.record_update", "resource.record_delete"}
	if fmt.Sprint(actions) != fmt.Sprint(want) {
		t.Fatalf("unexpected audit actions: %#v", actions)
	}
}

func TestTwoDifferentExampleResourcesUseTheSameEngine(t *testing.T) {
	router, _, token := setup(t)
	examples := []struct {
		code        string
		input       map[string]any
		assertField string
		assertValue any
	}{
		{code: "articles", input: map[string]any{"title": "Generic article", "body": "Long-form content", "views": 7, "published": true}, assertField: "title", assertValue: "Generic article"},
		{code: "products", input: map[string]any{"sku": "SKU-001", "name": "Generic product", "price_cents": 1299, "stock": 5, "metadata": map[string]any{"color": "blue"}}, assertField: "sku", assertValue: "SKU-001"},
	}
	for _, example := range examples {
		t.Run(example.code, func(t *testing.T) {
			created := do(router, http.MethodPost, "/api/v1/resources/"+example.code+"/records", example.input, token)
			if created.Code != 201 {
				t.Fatalf("create expected 201, got %d: %s", created.Code, created.Body.String())
			}
			listed := do(router, http.MethodGet, "/api/v1/resources/"+example.code+"/records", nil, token)
			if listed.Code != 200 {
				t.Fatalf("list expected 200, got %d: %s", listed.Code, listed.Body.String())
			}
			var payload struct {
				Data []map[string]any `json:"data"`
			}
			if err := json.Unmarshal(listed.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Data) != 1 || payload.Data[0][example.assertField] != example.assertValue {
				t.Fatalf("unexpected records: %#v", payload.Data)
			}
		})
	}
}
