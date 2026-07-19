package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/daqing/airway/app/middleware"
	"github.com/daqing/airway/app/modules/rbac"
	"github.com/daqing/airway/lib/repo"
	pluginsdk "github.com/daqing/airway/plugin"
	"github.com/gin-gonic/gin"
)

type Runtime struct {
	db            *repo.DB
	authorization *rbac.Service
	menusMu       sync.RWMutex
	menus         []pluginsdk.Menu
}

func NewRuntime(db *repo.DB, authorization *rbac.Service) *Runtime {
	return &Runtime{db: db, authorization: authorization, menus: make([]pluginsdk.Menu, 0)}
}

// Mount validates and registers all statically linked plugins under the
// authenticated /plugins/:name namespace.
func (r *Runtime) Mount(parent *gin.RouterGroup) error {
	for _, candidate := range pluginsdk.Registered() {
		manifest := candidate.Manifest()
		if err := manifest.Validate(pluginsdk.CoreVersion); err != nil {
			return err
		}
		group := parent.Group("/plugins/" + manifest.Name)
		ctx := &pluginContext{runtime: r, group: group, manifest: manifest, declared: map[string]bool{}}
		for _, permission := range manifest.Permissions {
			ctx.declared[permission.Code] = true
			if err := ctx.AddPermission(permission); err != nil {
				return fmt.Errorf("plugin %s permission %s: %w", manifest.Name, permission.Code, err)
			}
		}
		if err := candidate.Register(ctx); err != nil {
			return fmt.Errorf("register plugin %s: %w", manifest.Name, err)
		}
		if err := r.persist(context.Background(), manifest); err != nil {
			return fmt.Errorf("persist plugin %s: %w", manifest.Name, err)
		}
	}
	parent.GET("/plugin-menus", r.menuHandler)
	return nil
}
func (r *Runtime) Menus() []pluginsdk.Menu {
	r.menusMu.RLock()
	defer r.menusMu.RUnlock()
	return append([]pluginsdk.Menu{}, r.menus...)
}

func (r *Runtime) menuHandler(c *gin.Context) {
	admin, ok := middleware.CurrentAdmin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"data": nil, "error": gin.H{"code": "unauthenticated", "message": "authentication required"}})
		return
	}
	menus := make([]pluginsdk.Menu, 0)
	for _, menu := range r.Menus() {
		if menu.Permission == "" {
			menus = append(menus, menu)
			continue
		}
		allowed, err := r.authorization.Allowed(c, admin, menu.Permission)
		if err != nil {
			c.JSON(500, gin.H{"data": nil, "error": gin.H{"code": "authorization_failed", "message": "authorization check failed"}})
			return
		}
		if allowed {
			menus = append(menus, menu)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": menus, "error": nil})
}

func (r *Runtime) persist(ctx context.Context, manifest pluginsdk.Manifest) error {
	data, _ := json.Marshal(manifest)
	var count int
	if err := r.db.Conn().GetContext(ctx, &count, r.db.Conn().Rebind(`SELECT COUNT(*) FROM plugins WHERE name=?`), manifest.Name); err != nil {
		return err
	}
	now := time.Now().UTC()
	if count == 0 {
		_, err := r.db.Conn().ExecContext(ctx, r.db.Conn().Rebind(`INSERT INTO plugins (name,version,api_version,status,manifest_json,installed_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`), manifest.Name, manifest.Version, manifest.APIVersion, "registered", string(data), now, now, now)
		return err
	}
	_, err := r.db.Conn().ExecContext(ctx, r.db.Conn().Rebind(`UPDATE plugins SET version=?,api_version=?,status='registered',manifest_json=?,updated_at=? WHERE name=?`), manifest.Version, manifest.APIVersion, string(data), now, manifest.Name)
	return err
}

type pluginContext struct {
	runtime  *Runtime
	group    *gin.RouterGroup
	manifest pluginsdk.Manifest
	declared map[string]bool
}

func (c *pluginContext) Handle(method, path, permission string, handler gin.HandlerFunc) error {
	method = strings.ToUpper(strings.TrimSpace(method))
	if !map[string]bool{http.MethodGet: true, http.MethodPost: true, http.MethodPut: true, http.MethodPatch: true, http.MethodDelete: true}[method] {
		return fmt.Errorf("unsupported HTTP method %q", method)
	}
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "..") || path == "/" {
		return fmt.Errorf("plugin route must be a safe relative path")
	}
	if !c.declared[permission] {
		return fmt.Errorf("route permission %q is not declared in the manifest", permission)
	}
	if handler == nil {
		return fmt.Errorf("route handler is required")
	}
	c.group.Handle(method, path, middleware.RequirePermission(c.runtime.authorization, permission), handler)
	return nil
}
func (c *pluginContext) AddPermission(permission pluginsdk.Permission) error {
	if !c.declared[permission.Code] {
		return fmt.Errorf("permission %q is not declared in the manifest", permission.Code)
	}
	_, err := c.runtime.authorization.CreatePermission(context.Background(), permission.Code, permission.Name)
	if err == nil || err == rbac.ErrConflict {
		return nil
	}
	return err
}
func (c *pluginContext) AddMenu(menu pluginsdk.Menu) error {
	if strings.TrimSpace(menu.Code) == "" || strings.TrimSpace(menu.Label) == "" || !strings.HasPrefix(menu.Path, "/") {
		return fmt.Errorf("menu code, label, and absolute path are required")
	}
	if menu.Permission != "" && !c.declared[menu.Permission] {
		return fmt.Errorf("menu permission %q is not declared in the manifest", menu.Permission)
	}
	c.runtime.menusMu.Lock()
	defer c.runtime.menusMu.Unlock()
	c.runtime.menus = append(c.runtime.menus, menu)
	return nil
}
