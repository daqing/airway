package plugins_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
