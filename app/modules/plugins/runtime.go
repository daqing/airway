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
	"github.com/jmoiron/sqlx"
)

const (
	statusEnabled            = "enabled"
	statusDisabled           = "disabled"
	statusIncompatible       = "incompatible"
	statusDependencyError    = "dependency_error"
	statusMigrationFailed    = "migration_failed"
	statusRegistrationFailed = "registration_failed"
)

type Runtime struct {
	db            *repo.DB
	authorization *rbac.Service
	menusMu       sync.RWMutex
	menus         []pluginsdk.Menu
	manifests     map[string]pluginsdk.Manifest
}
type pluginRecord struct {
	Name         string    `db:"name" json:"name"`
	Version      string    `db:"version" json:"version"`
	APIVersion   string    `db:"api_version" json:"api_version"`
	Status       string    `db:"status" json:"status"`
	ErrorMessage *string   `db:"error_message" json:"error_message"`
	InstalledAt  time.Time `db:"installed_at" json:"installed_at"`
}

func NewRuntime(db *repo.DB, authorization *rbac.Service) *Runtime {
	return &Runtime{db: db, authorization: authorization, menus: []pluginsdk.Menu{}, manifests: map[string]pluginsdk.Manifest{}}
}

func (r *Runtime) Mount(parent *gin.RouterGroup) error {
	candidates := pluginsdk.Registered()
	for _, candidate := range candidates {
		r.manifests[candidate.Manifest().Name] = candidate.Manifest()
	}
	var failures []string
	for _, candidate := range candidates {
		manifest := candidate.Manifest()
		if err := manifest.Validate(pluginsdk.CoreVersion); err != nil {
			_ = r.persist(manifest, statusIncompatible, err.Error())
			failures = append(failures, err.Error())
			continue
		}
		if err := r.checkDependencies(manifest, false); err != nil {
			_ = r.persist(manifest, statusDependencyError, err.Error())
			failures = append(failures, err.Error())
			continue
		}
		group := parent.Group("/plugins/" + manifest.Name)
		ctx := &pluginContext{runtime: r, group: group, manifest: manifest, declared: map[string]bool{}, migrations: map[string]pluginsdk.Migration{}}
		for _, permission := range manifest.Permissions {
			ctx.declared[permission.Code] = true
			if err := ctx.AddPermission(permission); err != nil {
				_ = r.persist(manifest, statusRegistrationFailed, err.Error())
				failures = append(failures, err.Error())
				continue
			}
		}
		if err := candidate.Register(ctx); err != nil {
			_ = r.persist(manifest, statusRegistrationFailed, err.Error())
			failures = append(failures, err.Error())
			continue
		}
		if err := r.runMigrations(manifest, ctx.migrations); err != nil {
			_ = r.persist(manifest, statusMigrationFailed, err.Error())
			failures = append(failures, err.Error())
			continue
		}
		status := statusEnabled
		var previous string
		_ = r.db.Conn().Get(&previous, r.db.Conn().Rebind(`SELECT status FROM plugins WHERE name=?`), manifest.Name)
		if previous == statusDisabled {
			status = statusDisabled
		}
		if err := r.persist(manifest, status, ""); err != nil {
			failures = append(failures, err.Error())
		}
	}
	parent.GET("/plugin-menus", r.menuHandler)
	parent.GET("/plugins", middleware.RequireSuperAdmin(), r.listHandler)
	parent.POST("/plugins/:name/enable", middleware.RequireSuperAdmin(), r.enableHandler)
	parent.POST("/plugins/:name/disable", middleware.RequireSuperAdmin(), r.disableHandler)
	if len(failures) > 0 {
		return fmt.Errorf("plugin startup failures: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (r *Runtime) checkDependencies(manifest pluginsdk.Manifest, requireEnabled bool) error {
	for _, dependency := range manifest.Dependencies {
		candidate, exists := r.manifests[dependency.Name]
		if !exists {
			return fmt.Errorf("missing dependency %s %s", dependency.Name, dependency.Version)
		}
		matches, err := pluginsdk.Satisfies(candidate.Version, dependency.Version)
		if err != nil || !matches {
			return fmt.Errorf("dependency conflict: %s requires %s, found %s", dependency.Name, dependency.Version, candidate.Version)
		}
		if requireEnabled {
			var status string
			if err := r.db.Conn().Get(&status, r.db.Conn().Rebind(`SELECT status FROM plugins WHERE name=?`), dependency.Name); err != nil || status != statusEnabled {
				return fmt.Errorf("dependency %s is not enabled", dependency.Name)
			}
		}
	}
	return nil
}
func (r *Runtime) runMigrations(manifest pluginsdk.Manifest, registered map[string]pluginsdk.Migration) error {
	for _, name := range manifest.Migrations {
		migration, ok := registered[name]
		if !ok {
			return fmt.Errorf("declared migration %s was not registered", name)
		}
		err := repo.Tx(r.db, func(tx *sqlx.Tx) error {
			var count int
			if err := tx.Get(&count, tx.Rebind(`SELECT COUNT(*) FROM plugin_migrations WHERE plugin_name=? AND migration=?`), manifest.Name, name); err != nil {
				return err
			}
			if count > 0 {
				return nil
			}
			if migration.Up == nil {
				return fmt.Errorf("migration %s has no Up callback", name)
			}
			if err := migration.Up(context.Background(), tx); err != nil {
				return fmt.Errorf("migration %s failed: %w", name, err)
			}
			_, err := tx.Exec(tx.Rebind(`INSERT INTO plugin_migrations (plugin_name,migration,applied_at) VALUES (?,?,?)`), manifest.Name, name, time.Now().UTC())
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func (r *Runtime) persist(manifest pluginsdk.Manifest, status, message string) error {
	data, _ := json.Marshal(manifest)
	now := time.Now().UTC()
	var count int
	if err := r.db.Conn().Get(&count, r.db.Conn().Rebind(`SELECT COUNT(*) FROM plugins WHERE name=?`), manifest.Name); err != nil {
		return err
	}
	var errorValue any
	if message != "" {
		errorValue = message
	}
	var enabledAt, disabledAt any
	if status == statusEnabled {
		enabledAt = now
	}
	if status == statusDisabled {
		disabledAt = now
	}
	if count == 0 {
		_, err := r.db.Conn().Exec(r.db.Conn().Rebind(`INSERT INTO plugins (name,version,api_version,status,manifest_json,error_message,enabled_at,disabled_at,installed_at,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`), manifest.Name, manifest.Version, manifest.APIVersion, status, string(data), errorValue, enabledAt, disabledAt, now, now, now)
		return err
	}
	_, err := r.db.Conn().Exec(r.db.Conn().Rebind(`UPDATE plugins SET version=?,api_version=?,status=?,manifest_json=?,error_message=?,enabled_at=COALESCE(?,enabled_at),disabled_at=COALESCE(?,disabled_at),updated_at=? WHERE name=?`), manifest.Version, manifest.APIVersion, status, string(data), errorValue, enabledAt, disabledAt, now, manifest.Name)
	return err
}

func (r *Runtime) Menus() []pluginsdk.Menu {
	r.menusMu.RLock()
	defer r.menusMu.RUnlock()
	return append([]pluginsdk.Menu{}, r.menus...)
}
func (r *Runtime) menuHandler(c *gin.Context) {
	admin, ok := middleware.CurrentAdmin(c)
	if !ok {
		fail(c, 401, "unauthenticated", "authentication required")
		return
	}
	menus := []pluginsdk.Menu{}
	for _, menu := range r.Menus() {
		name := pluginNameForPath(menu.Path)
		if name != "" && !r.isEnabled(name) {
			continue
		}
		if menu.Permission == "" {
			menus = append(menus, menu)
			continue
		}
		allowed, err := r.authorization.Allowed(c, admin, menu.Permission)
		if err != nil {
			fail(c, 500, "authorization_failed", "authorization check failed")
			return
		}
		if allowed {
			menus = append(menus, menu)
		}
	}
	c.JSON(200, gin.H{"data": menus, "error": nil})
}
func (r *Runtime) listHandler(c *gin.Context) {
	items := []pluginRecord{}
	if err := r.db.Conn().Select(&items, `SELECT name,version,api_version,status,error_message,installed_at FROM plugins ORDER BY name`); err != nil {
		fail(c, 500, "internal_error", err.Error())
		return
	}
	c.JSON(200, gin.H{"data": items, "error": nil})
}
func (r *Runtime) enableHandler(c *gin.Context) {
	name := c.Param("name")
	manifest, ok := r.manifests[name]
	if !ok {
		fail(c, 404, "plugin_not_found", "plugin is not compiled into this application")
		return
	}
	var record pluginRecord
	if err := r.db.Conn().Get(&record, r.db.Conn().Rebind(`SELECT name,version,api_version,status,error_message,installed_at FROM plugins WHERE name=?`), name); err != nil {
		fail(c, 404, "plugin_not_found", "plugin is not registered")
		return
	}
	if record.Status == statusIncompatible || record.Status == statusDependencyError || record.Status == statusMigrationFailed || record.Status == statusRegistrationFailed {
		message := "plugin cannot be enabled"
		if record.ErrorMessage != nil {
			message = *record.ErrorMessage
		}
		fail(c, 409, "plugin_not_ready", message)
		return
	}
	if err := r.checkDependencies(manifest, true); err != nil {
		fail(c, 409, "dependency_conflict", err.Error())
		return
	}
	_ = r.persist(manifest, statusEnabled, "")
	c.JSON(200, gin.H{"data": gin.H{"name": name, "status": statusEnabled}, "error": nil})
}
func (r *Runtime) disableHandler(c *gin.Context) {
	name := c.Param("name")
	manifest, ok := r.manifests[name]
	if !ok {
		fail(c, 404, "plugin_not_found", "plugin is not compiled into this application")
		return
	}
	for dependent, candidate := range r.manifests {
		for _, dependency := range candidate.Dependencies {
			if dependency.Name == name && r.isEnabled(dependent) {
				fail(c, 409, "dependency_conflict", fmt.Sprintf("plugin %s is required by enabled plugin %s", name, dependent))
				return
			}
		}
	}
	_ = r.persist(manifest, statusDisabled, "")
	c.JSON(200, gin.H{"data": gin.H{"name": name, "status": statusDisabled}, "error": nil})
}
func (r *Runtime) isEnabled(name string) bool {
	var status string
	return r.db.Conn().Get(&status, r.db.Conn().Rebind(`SELECT status FROM plugins WHERE name=?`), name) == nil && status == statusEnabled
}
func (r *Runtime) enabledMiddleware(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !r.isEnabled(name) {
			fail(c, 503, "plugin_disabled", "plugin is not enabled")
			c.Abort()
			return
		}
		c.Next()
	}
}

type pluginContext struct {
	runtime    *Runtime
	group      *gin.RouterGroup
	manifest   pluginsdk.Manifest
	declared   map[string]bool
	migrations map[string]pluginsdk.Migration
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
	c.group.Handle(method, path, c.runtime.enabledMiddleware(c.manifest.Name), middleware.RequirePermission(c.runtime.authorization, permission), handler)
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
func (c *pluginContext) AddMigration(migration pluginsdk.Migration) error {
	declared := false
	for _, name := range c.manifest.Migrations {
		if name == migration.Name {
			declared = true
			break
		}
	}
	if !declared {
		return fmt.Errorf("migration %q is not declared in the manifest", migration.Name)
	}
	if _, exists := c.migrations[migration.Name]; exists {
		return fmt.Errorf("migration %q is already registered", migration.Name)
	}
	c.migrations[migration.Name] = migration
	return nil
}
func pluginNameForPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "plugins" {
		return parts[1]
	}
	return ""
}
func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"data": nil, "error": gin.H{"code": code, "message": message}})
}
