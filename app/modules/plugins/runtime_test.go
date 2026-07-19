package plugins_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/daqing/airway/app/middleware"
	"github.com/daqing/airway/app/modules/identity"
	pluginruntime "github.com/daqing/airway/app/modules/plugins"
	"github.com/daqing/airway/app/modules/rbac"
	_ "github.com/daqing/airway/db/migrate"
	"github.com/daqing/airway/lib/migrate/dialect"
	"github.com/daqing/airway/lib/migrate/schema"
	"github.com/daqing/airway/lib/repo"
	pluginsdk "github.com/daqing/airway/plugin"
	"github.com/gin-gonic/gin"
)

type registeredPlugin struct{}

func (registeredPlugin) Manifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{APIVersion: pluginsdk.APIVersion, Name: "hello-plugin", Version: "1.0.0", Core: ">=0.1.0 <0.2.0", Entry: "helloplugin.Register", Permissions: []pluginsdk.Permission{{Code: "hello:read", Name: "Read hello"}}}
}
func (registeredPlugin) Register(ctx pluginsdk.Context) error {
	if err := ctx.AddMenu(pluginsdk.Menu{Code: "hello", Label: "Hello", Path: "/plugins/hello", Permission: "hello:read"}); err != nil {
		return err
	}
	return ctx.Handle(http.MethodGet, "/status", "hello:read", func(c *gin.Context) { c.JSON(200, gin.H{"data": "ok", "error": nil}) })
}

func TestRuntimeRegistersManifestPermissionMenuAndProtectedRoute(t *testing.T) {
	if err := pluginsdk.Register(registeredPlugin{}); err != nil {
		t.Fatal(err)
	}
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
	gin.SetMode(gin.TestMode)
	router := gin.New()
	runtime := pluginruntime.NewRuntime(db, rbac.NewService(db))
	if err := runtime.Mount(router.Group("/api/v1")); err != nil {
		t.Fatal(err)
	}
	var pluginCount, permissionCount int
	if err := db.Conn().Get(&pluginCount, `SELECT COUNT(*) FROM plugins WHERE name='hello-plugin' AND version='1.0.0' AND api_version='airway.dev/v1'`); err != nil || pluginCount != 1 {
		t.Fatalf("persisted plugin count=%d err=%v", pluginCount, err)
	}
	if err := db.Conn().Get(&permissionCount, `SELECT COUNT(*) FROM permissions WHERE code='hello:read'`); err != nil || permissionCount != 1 {
		t.Fatalf("permission count=%d err=%v", permissionCount, err)
	}
	menus := runtime.Menus()
	if len(menus) != 1 || menus[0].Code != "hello" {
		t.Fatalf("menus=%#v", menus)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/hello-plugin/status", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("protected plugin route expected 401, got %d: %s", response.Code, response.Body.String())
	}
}

type lifecyclePlugin struct {
	manifest pluginsdk.Manifest
	register func(pluginsdk.Context) error
}

func (p lifecyclePlugin) Manifest() pluginsdk.Manifest { return p.manifest }
func (p lifecyclePlugin) Register(ctx pluginsdk.Context) error {
	if p.register != nil {
		return p.register(ctx)
	}
	return nil
}
func lifecycleManifest(name string) pluginsdk.Manifest {
	code := strings.ReplaceAll(name, "-", "_")
	return pluginsdk.Manifest{APIVersion: pluginsdk.APIVersion, Name: name, Version: "1.0.0", Core: ">=0.1.0 <0.2.0", Entry: name + ".Register", Permissions: []pluginsdk.Permission{{Code: code + ":read", Name: "Read " + name}}}
}

func TestCompatibilityDependencyMigrationAndEnableDisableFeedback(t *testing.T) {
	good := lifecyclePlugin{manifest: lifecycleManifest("lifecycle-good"), register: func(ctx pluginsdk.Context) error {
		return ctx.Handle(http.MethodGet, "/status", "lifecycle_good:read", func(c *gin.Context) { c.JSON(200, gin.H{"data": "ok", "error": nil}) })
	}}
	incompatibleManifest := lifecycleManifest("lifecycle-incompatible")
	incompatibleManifest.Core = ">=9.0.0"
	missingManifest := lifecycleManifest("lifecycle-missing-dep")
	missingManifest.Dependencies = []pluginsdk.Dependency{{Name: "not-installed", Version: ">=1.0.0"}}
	migrationManifest := lifecycleManifest("lifecycle-migration")
	migrationManifest.Migrations = []string{"001_fail"}
	migrationFailure := lifecyclePlugin{manifest: migrationManifest, register: func(ctx pluginsdk.Context) error {
		return ctx.AddMigration(pluginsdk.Migration{Name: "001_fail", Up: func(ctx context.Context, executor pluginsdk.Executor) error {
			if _, err := executor.ExecContext(ctx, `CREATE TABLE should_rollback (id INTEGER)`); err != nil {
				return err
			}
			return errors.New("deliberate migration failure")
		}})
	}}
	for _, candidate := range []pluginsdk.Plugin{good, lifecyclePlugin{manifest: incompatibleManifest}, lifecyclePlugin{manifest: missingManifest}, migrationFailure} {
		if err := pluginsdk.Register(candidate); err != nil {
			t.Fatal(err)
		}
	}
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
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1")
	group.Use(middleware.Authenticate(identities))
	runtime := pluginruntime.NewRuntime(db, rbac.NewService(db))
	if err := runtime.Mount(group); err == nil {
		t.Fatal("expected aggregate startup feedback")
	}
	statuses := map[string]string{}
	rows, err := db.Conn().Queryx(`SELECT name,status FROM plugins`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name, status string
		if err := rows.Scan(&name, &status); err != nil {
			t.Fatal(err)
		}
		statuses[name] = status
	}
	_ = rows.Close()
	for name, want := range map[string]string{"lifecycle-good": "enabled", "lifecycle-incompatible": "incompatible", "lifecycle-missing-dep": "dependency_error", "lifecycle-migration": "migration_failed"} {
		if statuses[name] != want {
			t.Fatalf("%s status=%q want=%q", name, statuses[name], want)
		}
	}
	var rollbackTableCount int
	if err := db.Conn().Get(&rollbackTableCount, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='should_rollback'`); err != nil || rollbackTableCount != 0 {
		t.Fatalf("failed migration was not rolled back: count=%d err=%v", rollbackTableCount, err)
	}
	call := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.AddCookie(&http.Cookie{Name: "airway_session", Value: token})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	if response := call(http.MethodGet, "/api/v1/plugins/lifecycle-good/status"); response.Code != 200 {
		t.Fatalf("enabled route: %d %s", response.Code, response.Body.String())
	}
	if response := call(http.MethodPost, "/api/v1/plugins/lifecycle-good/disable"); response.Code != 200 {
		t.Fatalf("disable: %d %s", response.Code, response.Body.String())
	}
	if response := call(http.MethodGet, "/api/v1/plugins/lifecycle-good/status"); response.Code != 503 {
		t.Fatalf("disabled route expected 503, got %d: %s", response.Code, response.Body.String())
	}
	if response := call(http.MethodPost, "/api/v1/plugins/lifecycle-good/enable"); response.Code != 200 {
		t.Fatalf("enable: %d %s", response.Code, response.Body.String())
	}
	if response := call(http.MethodGet, "/api/v1/plugins/lifecycle-good/status"); response.Code != 200 {
		t.Fatalf("re-enabled route: %d %s", response.Code, response.Body.String())
	}
}
