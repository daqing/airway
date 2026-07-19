package systeminfo

import (
	"net/http"
	"testing"

	pluginsdk "github.com/daqing/airway/plugin"
	"github.com/gin-gonic/gin"
)

type captureContext struct {
	permissions []pluginsdk.Permission
	menus       []pluginsdk.Menu
	method      string
	path        string
	permission  string
	handler     gin.HandlerFunc
}

func (c *captureContext) Handle(method, path, permission string, handler gin.HandlerFunc) error {
	c.method, c.path, c.permission, c.handler = method, path, permission, handler
	return nil
}
func (c *captureContext) AddPermission(permission pluginsdk.Permission) error {
	c.permissions = append(c.permissions, permission)
	return nil
}
func (c *captureContext) AddMenu(menu pluginsdk.Menu) error {
	c.menus = append(c.menus, menu)
	return nil
}
func (c *captureContext) AddMigration(migration pluginsdk.Migration) error { return nil }

func TestExamplePluginRegistersPermissionMenuAndAPI(t *testing.T) {
	candidate := Plugin{}
	if err := candidate.Manifest().Validate(pluginsdk.CoreVersion); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	ctx := &captureContext{}
	if err := candidate.Register(ctx); err != nil {
		t.Fatal(err)
	}
	if len(ctx.permissions) != 1 || ctx.permissions[0].Code != PermissionRead {
		t.Fatalf("permissions=%#v", ctx.permissions)
	}
	if len(ctx.menus) != 1 || ctx.menus[0].Path != "/plugins/system-info" {
		t.Fatalf("menus=%#v", ctx.menus)
	}
	if ctx.method != http.MethodGet || ctx.path != "/status" || ctx.permission != PermissionRead || ctx.handler == nil {
		t.Fatalf("route=%s %s permission=%s handler=%v", ctx.method, ctx.path, ctx.permission, ctx.handler != nil)
	}
}
