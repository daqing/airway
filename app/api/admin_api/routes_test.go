package admin_api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daqing/airway/app/api/admin_api"
	"github.com/daqing/airway/app/middleware"
	"github.com/daqing/airway/app/modules/identity"
	"github.com/daqing/airway/app/modules/rbac"
	_ "github.com/daqing/airway/db/migrate"
	"github.com/daqing/airway/lib/migrate/dialect"
	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

type permissionCase struct {
	name       string
	permission string
	method     string
	path       func(fixture) string
	body       func(fixture, int) any
	wantStatus int
}

type fixture struct {
	targetAdminID      int64
	targetRoleID       int64
	targetPermissionID int64
}

func setupPermissionTest(t *testing.T) (*gin.Engine, *repo.DB, *rbac.Service, identity.Admin, string, int64, fixture) {
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

	identityService := identity.NewService(db, 12*time.Hour)
	rbacService := rbac.NewService(db)
	actor, err := rbacService.CreateAdmin(context.Background(), "operator", "operator@example.com", "very-secure-password")
	if err != nil {
		t.Fatalf("create actor: %v", err)
	}
	target, err := rbacService.CreateAdmin(context.Background(), "target", "target@example.com", "very-secure-password")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	authRole, err := rbacService.CreateRole(context.Background(), "api-tester", "API tester")
	if err != nil {
		t.Fatalf("create authorization role: %v", err)
	}
	targetRole, err := rbacService.CreateRole(context.Background(), "target-role", "Target role")
	if err != nil {
		t.Fatalf("create target role: %v", err)
	}
	targetPermission, err := rbacService.CreatePermission(context.Background(), "target:action", "Target action")
	if err != nil {
		t.Fatalf("create target permission: %v", err)
	}
	if err := rbacService.AssignRoles(context.Background(), actor.ID, []int64{authRole.ID}); err != nil {
		t.Fatalf("assign actor role: %v", err)
	}
	_, token, _, err := identityService.Login(context.Background(), actor.Login, "very-secure-password", "test", "test-request")
	if err != nil {
		t.Fatalf("login actor: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	protected := router.Group("/api/v1")
	protected.Use(middleware.Authenticate(identityService))
	admin_api.Routes(protected, rbacService)
	return router, db, rbacService, actor.Admin, token, authRole.ID, fixture{
		targetAdminID: target.ID, targetRoleID: targetRole.ID, targetPermissionID: targetPermission.ID,
	}
}

func performRequest(router *gin.Engine, method, path string, body any, token string) *httptest.ResponseRecorder {
	var encoded []byte
	if body != nil {
		encoded, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-request")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "airway_session", Value: token})
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func TestEveryManagementAPIRouteRequiresItsDeclaredPermission(t *testing.T) {
	router, _, service, _, token, authRoleID, f := setupPermissionTest(t)
	cases := []permissionCase{
		{name: "list admins", permission: "admins:read", method: http.MethodGet, path: func(f fixture) string { return "/api/v1/admins" }, wantStatus: 200},
		{name: "create admin", permission: "admins:create", method: http.MethodPost, path: func(f fixture) string { return "/api/v1/admins" }, body: func(_ fixture, n int) any {
			return map[string]any{"login": fmt.Sprintf("created-%d", n), "email": fmt.Sprintf("created-%d@example.com", n), "password": "very-secure-password"}
		}, wantStatus: 201},
		{name: "update admin", permission: "admins:update", method: http.MethodPatch, path: func(f fixture) string { return fmt.Sprintf("/api/v1/admins/%d", f.targetAdminID) }, body: func(_ fixture, _ int) any {
			return map[string]any{"email": "updated-target@example.com", "status": "active"}
		}, wantStatus: 200},
		{name: "list admin roles", permission: "admins:read", method: http.MethodGet, path: func(f fixture) string { return fmt.Sprintf("/api/v1/admins/%d/roles", f.targetAdminID) }, wantStatus: 200},
		{name: "assign admin roles", permission: "admins:assign_roles", method: http.MethodPut, path: func(f fixture) string { return fmt.Sprintf("/api/v1/admins/%d/roles", f.targetAdminID) }, body: func(f fixture, _ int) any { return map[string]any{"role_ids": []int64{f.targetRoleID}} }, wantStatus: 200},
		{name: "list roles", permission: "roles:read", method: http.MethodGet, path: func(f fixture) string { return "/api/v1/roles" }, wantStatus: 200},
		{name: "create role", permission: "roles:create", method: http.MethodPost, path: func(f fixture) string { return "/api/v1/roles" }, body: func(_ fixture, n int) any {
			return map[string]any{"code": fmt.Sprintf("created-role-%d", n), "name": "Created role"}
		}, wantStatus: 201},
		{name: "list role permissions", permission: "roles:read", method: http.MethodGet, path: func(f fixture) string { return fmt.Sprintf("/api/v1/roles/%d/permissions", f.targetRoleID) }, wantStatus: 200},
		{name: "assign role permissions", permission: "roles:assign_permissions", method: http.MethodPut, path: func(f fixture) string { return fmt.Sprintf("/api/v1/roles/%d/permissions", f.targetRoleID) }, body: func(f fixture, _ int) any { return map[string]any{"permission_ids": []int64{f.targetPermissionID}} }, wantStatus: 200},
		{name: "list permissions", permission: "permissions:read", method: http.MethodGet, path: func(f fixture) string { return "/api/v1/permissions" }, wantStatus: 200},
		{name: "create permission", permission: "permissions:create", method: http.MethodPost, path: func(f fixture) string { return "/api/v1/permissions" }, body: func(_ fixture, n int) any {
			return map[string]any{"code": fmt.Sprintf("created-%d:action", n), "name": "Created permission"}
		}, wantStatus: 201},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := any(nil)
			if tc.body != nil {
				body = tc.body(f, i)
			}
			if err := service.AssignPermissions(context.Background(), authRoleID, nil); err != nil {
				t.Fatalf("clear permissions: %v", err)
			}

			unauthenticated := performRequest(router, tc.method, tc.path(f), body, "")
			if unauthenticated.Code != http.StatusUnauthorized {
				t.Fatalf("without session: expected 401, got %d: %s", unauthenticated.Code, unauthenticated.Body.String())
			}
			forbidden := performRequest(router, tc.method, tc.path(f), body, token)
			if forbidden.Code != http.StatusForbidden {
				t.Fatalf("without %s: expected 403, got %d: %s", tc.permission, forbidden.Code, forbidden.Body.String())
			}

			permission, err := service.CreatePermission(context.Background(), tc.permission, tc.name)
			if err != nil {
				permissions, listErr := service.ListPermissions(context.Background())
				if listErr != nil {
					t.Fatal(listErr)
				}
				for _, existing := range permissions {
					if existing.Code == tc.permission {
						permission = existing
						break
					}
				}
			}
			if permission.ID == 0 {
				t.Fatalf("permission %s was not created", tc.permission)
			}
			if err := service.AssignPermissions(context.Background(), authRoleID, []int64{permission.ID}); err != nil {
				t.Fatalf("grant permission: %v", err)
			}
			allowed := performRequest(router, tc.method, tc.path(f), body, token)
			if allowed.Code != tc.wantStatus {
				t.Fatalf("with %s: expected %d, got %d: %s", tc.permission, tc.wantStatus, allowed.Code, allowed.Body.String())
			}
		})
	}
}

func TestPermissionChangesTakeEffectForTheCurrentSession(t *testing.T) {
	router, _, service, actor, token, authRoleID, _ := setupPermissionTest(t)
	permission, err := service.CreatePermission(context.Background(), "admins:read", "Read administrators")
	if err != nil {
		t.Fatalf("create permission: %v", err)
	}

	if err := service.AssignPermissions(context.Background(), authRoleID, []int64{permission.ID}); err != nil {
		t.Fatalf("grant permission: %v", err)
	}
	response := performRequest(router, http.MethodGet, "/api/v1/admins", nil, token)
	if response.Code != http.StatusOK {
		t.Fatalf("after grant: expected 200, got %d: %s", response.Code, response.Body.String())
	}

	// Revoking a role permission must affect the already-issued session.
	if err := service.AssignPermissions(context.Background(), authRoleID, nil); err != nil {
		t.Fatalf("revoke permission: %v", err)
	}
	response = performRequest(router, http.MethodGet, "/api/v1/admins", nil, token)
	if response.Code != http.StatusForbidden {
		t.Fatalf("after permission revocation: expected 403, got %d: %s", response.Code, response.Body.String())
	}

	if err := service.AssignPermissions(context.Background(), authRoleID, []int64{permission.ID}); err != nil {
		t.Fatalf("restore permission: %v", err)
	}
	response = performRequest(router, http.MethodGet, "/api/v1/admins", nil, token)
	if response.Code != http.StatusOK {
		t.Fatalf("after permission restoration: expected 200, got %d: %s", response.Code, response.Body.String())
	}

	// Removing the role itself must also affect the current session immediately.
	if err := service.AssignRoles(context.Background(), actor.ID, nil); err != nil {
		t.Fatalf("remove actor role: %v", err)
	}
	response = performRequest(router, http.MethodGet, "/api/v1/admins", nil, token)
	if response.Code != http.StatusForbidden {
		t.Fatalf("after role removal: expected 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestDisablingAdminInvalidatesExistingSession(t *testing.T) {
	router, _, service, actor, token, authRoleID, _ := setupPermissionTest(t)
	permission, err := service.CreatePermission(context.Background(), "admins:read", "Read administrators")
	if err != nil {
		t.Fatalf("create permission: %v", err)
	}
	if err := service.AssignPermissions(context.Background(), authRoleID, []int64{permission.ID}); err != nil {
		t.Fatalf("grant permission: %v", err)
	}

	response := performRequest(router, http.MethodGet, "/api/v1/admins", nil, token)
	if response.Code != http.StatusOK {
		t.Fatalf("before disable: expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if _, err := service.UpdateAdmin(context.Background(), actor.ID, actor.Email, "disabled"); err != nil {
		t.Fatalf("disable administrator: %v", err)
	}

	// Authentication rejects the old token before authorization is evaluated.
	response = performRequest(router, http.MethodGet, "/api/v1/admins", nil, token)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("after disable: expected 401, got %d: %s", response.Code, response.Body.String())
	}
}

func TestLastActiveSuperAdminCannotBeDisabled(t *testing.T) {
	router, db, _, actor, token, _, f := setupPermissionTest(t)
	if _, err := db.Conn().Exec(db.Conn().Rebind(`UPDATE admins SET super_admin=? WHERE id=?`), true, actor.ID); err != nil {
		t.Fatalf("promote actor: %v", err)
	}
	body := map[string]any{"email": actor.Email, "status": "disabled"}
	path := fmt.Sprintf("/api/v1/admins/%d", actor.ID)

	response := performRequest(router, http.MethodPatch, path, body, token)
	if response.Code != http.StatusConflict {
		t.Fatalf("last super administrator: expected 409, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != "last_super_admin" {
		t.Fatalf("expected last_super_admin error, got %q", payload.Error.Code)
	}
	var status string
	if err := db.Conn().Get(&status, db.Conn().Rebind(`SELECT status FROM admins WHERE id=?`), actor.ID); err != nil || status != "active" {
		t.Fatalf("protected administrator status=%q err=%v", status, err)
	}

	// Once another active super administrator exists, disabling this one is safe.
	if _, err := db.Conn().Exec(db.Conn().Rebind(`UPDATE admins SET super_admin=? WHERE id=?`), true, f.targetAdminID); err != nil {
		t.Fatalf("promote second super administrator: %v", err)
	}
	response = performRequest(router, http.MethodPatch, path, body, token)
	if response.Code != http.StatusOK {
		t.Fatalf("with second super administrator: expected 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestOnlySuperAdminCanEditRoleNameAndCode(t *testing.T) {
	router, db, service, actor, token, authRoleID, f := setupPermissionTest(t)
	permission, err := service.CreatePermission(context.Background(), "roles:update", "Update roles")
	if err != nil {
		t.Fatalf("create roles:update permission: %v", err)
	}
	if err := service.AssignPermissions(context.Background(), authRoleID, []int64{permission.ID}); err != nil {
		t.Fatalf("grant roles:update: %v", err)
	}
	path := fmt.Sprintf("/api/v1/roles/%d", f.targetRoleID)
	body := map[string]any{"code": "reviewer", "name": "Reviewer"}

	response := performRequest(router, http.MethodPatch, path, body, token)
	if response.Code != http.StatusForbidden {
		t.Fatalf("regular administrator with roles:update: expected 403, got %d: %s", response.Code, response.Body.String())
	}

	if _, err := db.Conn().Exec(db.Conn().Rebind(`UPDATE admins SET super_admin=? WHERE id=?`), true, actor.ID); err != nil {
		t.Fatalf("promote actor: %v", err)
	}
	response = performRequest(router, http.MethodPatch, path, body, token)
	if response.Code != http.StatusOK {
		t.Fatalf("super administrator: expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var role rbac.Role
	if err := json.Unmarshal(response.Body.Bytes(), &struct {
		Data *rbac.Role `json:"data"`
	}{Data: &role}); err != nil {
		t.Fatalf("decode updated role: %v", err)
	}
	if role.Code != "reviewer" || role.Name != "Reviewer" {
		t.Fatalf("unexpected updated role: %#v", role)
	}
}

func TestOnlySuperAdminCanEditPermissionNameAndCode(t *testing.T) {
	router, db, service, actor, token, authRoleID, f := setupPermissionTest(t)
	permission, err := service.CreatePermission(context.Background(), "permissions:update", "Update permissions")
	if err != nil {
		t.Fatalf("create permissions:update permission: %v", err)
	}
	if err := service.AssignPermissions(context.Background(), authRoleID, []int64{permission.ID}); err != nil {
		t.Fatalf("grant permissions:update: %v", err)
	}
	path := fmt.Sprintf("/api/v1/permissions/%d", f.targetPermissionID)
	body := map[string]any{"code": "reviews:approve", "name": "Approve reviews"}

	response := performRequest(router, http.MethodPatch, path, body, token)
	if response.Code != http.StatusForbidden {
		t.Fatalf("regular administrator with permissions:update: expected 403, got %d: %s", response.Code, response.Body.String())
	}

	if _, err := db.Conn().Exec(db.Conn().Rebind(`UPDATE admins SET super_admin=? WHERE id=?`), true, actor.ID); err != nil {
		t.Fatalf("promote actor: %v", err)
	}
	response = performRequest(router, http.MethodPatch, path, body, token)
	if response.Code != http.StatusOK {
		t.Fatalf("super administrator: expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data rbac.Permission `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode updated permission: %v", err)
	}
	if payload.Data.Code != "reviews:approve" || payload.Data.Name != "Approve reviews" {
		t.Fatalf("unexpected updated permission: %#v", payload.Data)
	}
}
