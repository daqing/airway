package cmd

// Templates for `airway admin:generate`. Per-table files (models, resources,
// pages, islands) self-register through init(), so re-runs only add new
// files; the once-only files (registry, routes, auth, layout) are written
// only when absent.

const adminModelTemplate = `package models

import (
	"time"
{{if .HasRefs}}	"github.com/daqing/airway/lib/repo"
{{end}}	airwaysql "github.com/daqing/airway/lib/sql"
)

type {{.Name}} struct {
	ID        airwaysql.IdType ` + "`db:\"id\" json:\"id\"`" + `
{{range .Fields}}	{{.Name}} {{.GoType}} ` + "`db:\"{{.JSON}}\" json:\"{{.JSON}}\"`" + `
{{end}}	CreatedAt time.Time ` + "`db:\"created_at\" json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`db:\"updated_at\" json:\"updated_at\"`" + `
}

func ({{.Name}}) TableName() string {
	return "{{.SlugPlural}}"
}
{{if .HasRefs}}
// Relations maps every references field to a belongs-to association.
func ({{.Name}}) Relations() map[string]repo.Relation {
	return map[string]repo.Relation{
{{range .Fields}}{{if eq .Kind "references"}}		"{{.DisplayName}}": repo.NewBelongsTo({{.RefGoName}}{}, "{{.Name}}"),
{{end}}{{end}}	}
}
{{end}}
func init() {
	registerREPLModel("{{.Name}}", {{.Name}}{})
}
`

const adminAuthUserModelTemplate = `package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

// AdminUser signs in to the generated admin panel (see admin_api).
// Role is one of admin, editor (default — read/write) or viewer (read-only).
type AdminUser struct {
	ID             airwaysql.IdType ` + "`db:\"id\" json:\"id\"`" + `
	Username       string           ` + "`db:\"username\" json:\"username\"`" + `
	PasswordDigest string           ` + "`db:\"password_digest\" json:\"-\"`" + `
	Role           string           ` + "`db:\"role\" json:\"role\"`" + `
	CreatedAt      time.Time        ` + "`db:\"created_at\" json:\"created_at\"`" + `
	UpdatedAt      time.Time        ` + "`db:\"updated_at\" json:\"updated_at\"`" + `
}

func (AdminUser) TableName() string {
	return "admin_users"
}

func init() {
	registerREPLModel("AdminUser", AdminUser{})
}
`

const adminAuthSessionModelTemplate = `package models

import (
	"time"

	airwaysql "github.com/daqing/airway/lib/sql"
)

// AdminSession is one signed-in admin browser: a random cookie token with an
// expiry, resolved by the AdminAuth middleware.
type AdminSession struct {
	ID          airwaysql.IdType ` + "`db:\"id\" json:\"id\"`" + `
	Token       string           ` + "`db:\"token\" json:\"token\"`" + `
	AdminUserID int64            ` + "`db:\"admin_user_id\" json:\"admin_user_id\"`" + `
	ExpiresAt   time.Time        ` + "`db:\"expires_at\" json:\"expires_at\"`" + `
	CreatedAt   time.Time        ` + "`db:\"created_at\" json:\"created_at\"`" + `
}

func (AdminSession) TableName() string {
	return "admin_sessions"
}

func init() {
	registerREPLModel("AdminSession", AdminSession{})
}
`

const adminMiddlewareTemplate = `package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"{{.Module}}/app/models"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"
)

// AdminSessionCookie carries the admin session token.
const AdminSessionCookie = "airway_admin_session"

// AdminAuth guards the admin pages and the admin API: page requests without
// a valid session are redirected to the sign-in page, API requests get a 401.
func AdminAuth(c *gin.Context) {
	token, err := c.Cookie(AdminSessionCookie)
	if err != nil || token == "" {
		adminReject(c)
		return
	}

	session, err := repo.FindOneBy[models.AdminSession](sql.H{"token": token})
	if err != nil || session == nil {
		adminReject(c)
		return
	}

	if !session.ExpiresAt.After(time.Now()) {
		_ = repo.DeleteByID[models.AdminSession](session.ID)
		adminReject(c)
		return
	}

	user, err := repo.FindByID[models.AdminUser](sql.IdType(session.AdminUserID))
	if err != nil || user == nil {
		adminReject(c)
		return
	}

	c.Set("admin_user", user)
	c.Set("admin_role", user.Role)
	c.Next()
}

func adminReject(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "unauthorized"})
		return
	}
	c.Redirect(http.StatusFound, utils.URLPrefix()+"/admin/login")
	c.Abort()
}

// adminRole returns the signed-in account's role; anything unknown counts as
// the least-privileged role.
func adminRole(c *gin.Context) string {
	if v, ok := c.Get("admin_role"); ok {
		if role, ok := v.(string); ok && role != "" {
			return role
		}
	}
	return "viewer"
}

// AdminRequireWrite guards mutating admin endpoints: viewer accounts are
// read-only, editor and admin may write.
func AdminRequireWrite(c *gin.Context) {
	if adminRole(c) == "viewer" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "message": "viewer accounts cannot modify data"})
		return
	}
	c.Next()
}

// AdminRequireAdmin guards admin-only pages such as the audit log; lesser
// roles are bounced back to the dashboard instead of getting a 403 page.
func AdminRequireAdmin(c *gin.Context) {
	if adminRole(c) != "admin" {
		c.Redirect(http.StatusFound, utils.URLPrefix()+"/admin")
		c.Abort()
		return
	}
	c.Next()
}
`

const adminRoutesTemplate = `package admin_api

import (
	"github.com/gin-gonic/gin"

	"{{.Module}}/app/middlewares"
)

// Routes mounts the whole admin module: server-rendered pages under /admin
// and the CRUD JSON API under /api/v1/admin. Reads sit behind AdminAuth,
// writes additionally behind AdminRequireWrite (viewer accounts are
// read-only), and the audit log behind AdminRequireAdmin. Resource routes
// and sidebar entries register themselves from the per-table
// *_resource.go init() hooks.
func Routes(r *gin.Engine) {
	r.GET("/admin/login", LoginPageAction)
	r.POST("/admin/login", LoginAction)
	r.POST("/admin/logout", LogoutAction)

	pages := r.Group("/admin", middlewares.AdminAuth)
	{
		pages.GET("", DashboardAction)
	}

	adminOnly := r.Group("/admin", middlewares.AdminAuth, middlewares.AdminRequireAdmin)
	{
		adminOnly.GET("/audit-log", AuditLogPageAction)
	}

	api := r.Group("/api/v1/admin", middlewares.AdminAuth)
	writes := r.Group("/api/v1/admin", middlewares.AdminAuth, middlewares.AdminRequireWrite)
	{
		writes.POST("/uploads", UploadAction)
	}

	for _, resource := range adminResources {
		resource.Mount(pages, api, writes)
	}
}
`

const adminRegistryTemplate = `package admin_api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"

	"{{.Module}}/app/models"
	"{{.Module}}/app/views/admin"
	"github.com/daqing/airway/lib/render"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
)

// AdminResource describes one generated CRUD resource. Re-running
// ` + "`airway admin:generate`" + ` only adds files, so the registry (and with it the
// sidebar and dashboard) picks up new tables automatically.
type AdminResource struct {
	Plural string
	Label  string
	Count  func() (int64, error)
	Page   func(c *gin.Context)
	Mount  func(pages *gin.RouterGroup, api *gin.RouterGroup, writes *gin.RouterGroup)
}

var adminResources []AdminResource

func registerAdminResource(resource AdminResource) {
	adminResources = append(adminResources, resource)
}

// adminNavItems builds the sidebar entries, marking the active page. The
// audit log only appears for admin accounts.
func adminNavItems(active string, role string) []admin.NavItem {
	items := []admin.NavItem{{"{{"}}Label: "Dashboard", Href: "/admin", Active: active == ""{{"}}"}}
	for _, resource := range adminResources {
		items = append(items, admin.NavItem{
			Label:  resource.Label,
			Href:   "/admin/" + resource.Plural,
			Active: resource.Plural == active,
		})
	}
	if role == "admin" {
		items = append(items, admin.NavItem{Label: "Audit Log", Href: "/admin/audit-log", Active: active == "@audit"})
	}
	return items
}

// adminPage wraps page content in the admin layout with the sidebar.
func adminPage(c *gin.Context, title string, active string, content templ.Component) templ.Component {
	return admin.AdminLayout(title, adminNavItems(active, c.GetString("admin_role")), content)
}

// DashboardAction renders the admin dashboard: one card per resource.
func DashboardAction(c *gin.Context) {
	cards := make([]admin.ResourceCard, 0, len(adminResources))
	for _, resource := range adminResources {
		count, err := resource.Count()
		if err != nil {
			render.Error(c, err)
			return
		}
		cards = append(cards, admin.ResourceCard{
			Label: resource.Label,
			Href:  "/admin/" + resource.Plural,
			Count: count,
		})
	}

	render.HTML(c, adminPage(c, "Dashboard", "", admin.Dashboard(cards)))
}

// adminListPaging reads the page/page_size query parameters with bounds.
func adminListPaging(c *gin.Context) (page int, pageSize int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// adminListOrder whitelists the sort column against the resource's own
// fields and clamps the direction, so request input never reaches SQL text.
func adminListOrder(c *gin.Context, allowed map[string]bool, fallback string) string {
	column := c.Query("sort")
	if !allowed[column] {
		return fallback
	}
	dir := strings.ToUpper(c.DefaultQuery("order", "asc"))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	return column + " " + dir
}

// adminQueryInt converts a query value to an exact-match filter value.
func adminQueryInt(v string) (any, bool) {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, false
	}
	return n, true
}

func adminQueryFloat(v string) (any, bool) {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil, false
	}
	return f, true
}

func adminQueryBool(v string) (any, bool) {
	switch v {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	}
	return nil, false
}

// adminAudit records a mutating action in admin_audit_logs. A failed audit
// write is logged, never fatal: it must not block the business action.
func adminAudit(c *gin.Context, action string, resource string, resourceID int64) {
	var userID int64
	username := ""
	if v, ok := c.Get("admin_user"); ok {
		if user, ok := v.(*models.AdminUser); ok {
			userID = int64(user.ID)
			username = user.Username
		}
	}

	if _, err := repo.InsertMap(repo.CurrentDB(), sql.Insert(sql.H{
		"admin_user_id":   userID,
		"admin_username":  username,
		"action":          action,
		"resource":        resource,
		"resource_id":     resourceID,
	}).Into("admin_audit_logs")); err != nil {
		fmt.Println("admin audit write failed:", err)
	}
}

// adminCSVCell neutralizes leading formula characters so spreadsheet apps do
// not interpret cell content as code, then quotes embedded separators.
func adminCSVCell(v string) string {
	if v != "" && strings.ContainsAny(v[:1], "=+-@\t") {
		v = "'" + v
	}
	if strings.ContainsAny(v, ",\"\n\r") {
		v = "\"" + strings.ReplaceAll(v, "\"", "\"\"") + "\""
	}
	return v
}

func adminCSVTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func adminCSVNullableString(v *string) string {
	if v == nil {
		return ""
	}
	return adminCSVCell(*v)
}

func adminCSVNullableInt(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}
`

const adminAuditActionsTemplate = `package admin_api

import (
	"time"

	"github.com/gin-gonic/gin"

	"{{.Module}}/app/views/admin"
	"github.com/daqing/airway/lib/render"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"
)

// AuditLogPageAction renders the read-only audit trail for admin accounts:
// the most recent recorded actions, newest first.
func AuditLogPageAction(c *gin.Context) {
	page, pageSize := adminListPaging(c)

	total, err := repo.Count(repo.CurrentDB(),
		sql.SelectColumns("count(*)").From("admin_audit_logs"))
	if err != nil {
		render.Error(c, err)
		return
	}

	rows, err := repo.FindMaps(repo.CurrentDB(),
		sql.Select("*").From("admin_audit_logs").OrderBy("id DESC").Limit(pageSize).Offset((page-1)*pageSize))
	if err != nil {
		render.Error(c, err)
		return
	}

	entries := make([]admin.AuditLogEntry, 0, len(rows))
	for _, row := range rows {
		entry := admin.AuditLogEntry{ResourceID: -1}
		if t, ok := row["created_at"].(time.Time); ok {
			entry.Time = t
		}
		if username, ok := row["admin_username"].(string); ok {
			entry.Username = username
		}
		if action, ok := row["action"].(string); ok {
			entry.Action = action
		}
		if resource, ok := row["resource"].(string); ok {
			entry.Resource = resource
		}
		if id, ok := row["resource_id"].(int64); ok {
			entry.ResourceID = id
		}
		entries = append(entries, entry)
	}

	pageCount := int(total)/pageSize + 1

	render.HTML(c, adminPage(c, "Audit Log", "@audit",
		admin.AuditLogPage(entries, page, pageCount, utils.URLPrefix())))
}
`

const adminAuthActionsTemplate = `package admin_api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"{{.Module}}/app/middlewares"
	"{{.Module}}/app/models"
	"{{.Module}}/app/views/admin"
	"github.com/daqing/airway/lib/ratelimit"
	"github.com/daqing/airway/lib/render"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"
)

// loginLimiter blunts credential stuffing: five failed attempts for the same
// IP and username pair lock it out for fifteen minutes.
var loginLimiter = ratelimit.New(5, 15*time.Minute)

const adminCSRFCookie = "airway_admin_csrf"

// issueCSRFToken sets a fresh double-submit token: it must come back both as
// a cookie and as the hidden form field on the sign-in POST.
func issueCSRFToken(c *gin.Context) string {
	token := utils.RandomHex(32)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     adminCSRFCookie,
		Value:    token,
		Path:     utils.URLPrefix() + "/admin",
		Expires:  time.Now().Add(12 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   !utils.AppConfig().IsLocal,
	})
	return token
}

// LoginPageAction serves the sign-in form.
func LoginPageAction(c *gin.Context) {
	render.HTML(c, admin.Login("", issueCSRFToken(c)))
}

type loginForm struct {
	Username  string ` + "`form:\"username\"`" + `
	Password  string ` + "`form:\"password\"`" + `
	CSRFToken string ` + "`form:\"csrf_token\"`" + `
}

// LoginAction verifies the CSRF token, rate-limits attempts per IP and
// username, and starts a cookie session on success.
func LoginAction(c *gin.Context) {
	var form loginForm
	if err := c.ShouldBind(&form); err != nil {
		render.HTML(c, admin.Login("Invalid username or password", issueCSRFToken(c)))
		return
	}

	if cookieToken, err := c.Cookie(adminCSRFCookie); err != nil || cookieToken == "" || cookieToken != form.CSRFToken {
		render.HTML(c, admin.Login("Your session expired — please try again", issueCSRFToken(c)))
		return
	}

	username := strings.TrimSpace(form.Username)
	limitKey := c.ClientIP() + "|" + username
	if !loginLimiter.Allowed(limitKey) {
		render.HTML(c, admin.Login("Too many attempts — try again in a few minutes", issueCSRFToken(c)))
		return
	}

	user, err := repo.FindOneBy[models.AdminUser](sql.H{"username": username})
	if err != nil || user == nil || !utils.ComparePassword(utils.PasswordDigest(user.PasswordDigest), form.Password) {
		loginLimiter.Fail(limitKey)
		render.HTML(c, admin.Login("Invalid username or password", issueCSRFToken(c)))
		return
	}
	loginLimiter.Reset(limitKey)

	token := utils.RandomHex(32)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if _, err := repo.CreateFrom[models.AdminSession](sql.H{
		"token":         token,
		"admin_user_id": user.ID,
		"expires_at":    expiresAt,
	}); err != nil {
		render.HTML(c, admin.Login("Could not sign in: "+err.Error(), issueCSRFToken(c)))
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middlewares.AdminSessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   !utils.AppConfig().IsLocal,
	})

	render.Found(c, utils.URLPrefix()+"/admin")
}

// LogoutAction deletes the session row and clears the cookie.
func LogoutAction(c *gin.Context) {
	if token, err := c.Cookie(middlewares.AdminSessionCookie); err == nil && token != "" {
		_ = repo.DeleteWhere[models.AdminSession](sql.H{"token": token})
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     middlewares.AdminSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   !utils.AppConfig().IsLocal,
	})

	render.Found(c, utils.URLPrefix()+"/admin/login")
}
`

const adminUploadsTemplate = `package admin_api

import (
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/lib/render"
	"github.com/daqing/airway/lib/storage"
	"github.com/daqing/airway/lib/utils"
)

// UploadAction stores an attachment through storage.Current() and returns
// its URL; it backs the attachment controls in the admin CRUD islands.
func UploadAction(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		render.ErrorMessage(c, ` + "`multipart form field \"file\" is required`" + `)
		return
	}

	src, err := file.Open()
	if err != nil {
		render.Error(c, err)
		return
	}
	defer src.Close()

	key := time.Now().Format("200601") + "/admin/" + utils.RandomHex(16) + strings.ToLower(filepath.Ext(file.Filename))

	obj := storage.Object{
		Reader:      src,
		Size:        file.Size,
		ContentType: adminContentType(file.Header.Get("Content-Type"), file.Filename),
	}
	if err := storage.Current().Put(c.Request.Context(), key, obj); err != nil {
		render.Error(c, err)
		return
	}

	url, err := storage.Current().URL(c.Request.Context(), key, 24*time.Hour)
	if err != nil {
		// The file is already stored; a URL failure must not fail the upload.
		url = key
	} else if strings.HasPrefix(url, "/") {
		url = utils.URLPrefix() + url
	}

	render.OK(c, gin.H{"key": key, "url": url})
}

func adminContentType(header, filename string) string {
	if header != "" {
		return header
	}
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}
`

const adminOpenAPITemplate = `package admin_api

import (
	"github.com/daqing/airway/lib/openapi"
)

func init() {
	openapi.Get("/admin", func(o *openapi.Operation) {
		o.Summary("Admin dashboard").Tag("admin")
		o.Respond(200, "text/html", openapi.Str()).
			Description("Server-rendered admin dashboard")
	})

	openapi.Get("/admin/login", func(o *openapi.Operation) {
		o.Summary("Admin sign-in page").Tag("admin")
		o.Respond(200, "text/html", openapi.Str()).
			Description("Server-rendered admin login form")
	})

	openapi.Post("/admin/login", func(o *openapi.Operation) {
		o.Summary("Admin sign-in").Tag("admin").
			Form(map[string]*openapi.Schema{
				"username": openapi.Str(),
				"password": openapi.Str(),
			}).
			OK()
	})

	openapi.Post("/admin/logout", func(o *openapi.Operation) {
		o.Summary("Admin sign-out").Tag("admin").
			OK()
	})

	openapi.Post("/api/v1/admin/uploads", func(o *openapi.Operation) {
		o.Summary("Upload an admin attachment").Tag("admin").
			FormUpload(map[string]*openapi.Schema{"file": openapi.File()}).
			OK(openapi.Obj(map[string]*openapi.Schema{
				"key": openapi.Str(),
				"url": openapi.Str(),
			}))
	})
}
`

const adminResourceTemplate = `// Code generated by ` + "`airway admin:generate`" + ` — edit freely; re-running the
// generator skips files that already exist (use --force to overwrite).
package admin_api

import (
	"encoding/csv"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"{{.Module}}/app/models"
	"{{.Module}}/app/views/admin"
	"github.com/daqing/airway/lib/openapi"
	"github.com/daqing/airway/lib/render"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
)

// {{.Name}}SortableFields whitelists the columns ` + "`sort`" + ` may target; request
// values outside this set fall back to the default ordering.
var {{.Name}}SortableFields = map[string]bool{
	"id":         true,
	"created_at": true,
	"updated_at": true,
{{range .Fields}}{{if .Form}}	"{{.JSON}}": true,
{{end}}{{end}}}

func init() {
	registerAdminResource(AdminResource{
		Plural: "{{.SlugPlural}}",
		Label:  "{{.Label}}",
		Count: func() (int64, error) {
			return repo.Count(repo.CurrentDB(),
				sql.SelectColumns("count(*)").From("{{.SlugPlural}}"){{if .SoftDeletable}}.Where(sql.Eq("deleted_at", nil)){{end}})
		},
		Page:  {{.Name}}PageAction,
		Mount: mount{{.Name}}Routes,
	})

	openapi.Get("/api/v1/admin/{{.SlugPlural}}", func(o *openapi.Operation) {
		o.Summary("Admin: list {{.NamePlural}}").Tag("admin").
			Query("page", openapi.Int(), "1-based page number").
			Query("page_size", openapi.Int(), "rows per page (1-100)").
			Query("q", openapi.Str(), "text search across string fields").
			Query("sort", openapi.Str(), "column to order by").
			Query("order", openapi.Str(), "asc or desc").
			OK(openapi.Obj(map[string]*openapi.Schema{
				"items": openapi.List[models.{{.Name}}](),
				"total": openapi.Int(),
			}))
	})

	openapi.Post("/api/v1/admin/{{.SlugPlural}}", func(o *openapi.Operation) {
		o.Summary("Admin: create a {{.Slug}}").Tag("admin").
			Body(openapi.Item[{{.Slug}}Params]()).
			OK(openapi.Item[models.{{.Name}}]())
	})

	openapi.Put("/api/v1/admin/{{.SlugPlural}}/{id}", func(o *openapi.Operation) {
		o.Summary("Admin: update a {{.Slug}}").Tag("admin").
			Path("id", openapi.Int(), "{{.Name}} id").
			Body(openapi.Item[{{.Slug}}Params]()).
			OK()
	})

	openapi.Delete("/api/v1/admin/{{.SlugPlural}}/{id}", func(o *openapi.Operation) {
		o.Summary("Admin: delete a {{.Slug}}").Tag("admin").
			Path("id", openapi.Int(), "{{.Name}} id").
			OK()
	})

	openapi.Get("/admin/{{.SlugPlural}}", func(o *openapi.Operation) {
		o.Summary("Admin {{.NamePlural}} page").Tag("admin")
		o.Respond(200, "text/html", openapi.Str()).
			Description("Server-rendered admin page hosting the {{.Slug}} CRUD island")
	})
}

func mount{{.Name}}Routes(pages *gin.RouterGroup, api *gin.RouterGroup, writes *gin.RouterGroup) {
	pages.GET("/{{.SlugPlural}}", {{.Name}}PageAction)

	api.GET("/{{.SlugPlural}}", {{.Name}}IndexAction)
	writes.POST("/{{.SlugPlural}}", {{.Name}}CreateAction)
	writes.PUT("/{{.SlugPlural}}/:id", {{.Name}}UpdateAction)
	writes.DELETE("/{{.SlugPlural}}/:id", {{.Name}}DestroyAction)
}

// {{.Name}}PageAction renders the admin {{.SlugPlural}} page.
func {{.Name}}PageAction(c *gin.Context) {
	render.HTML(c, adminPage(c, "{{.Label}}", "{{.SlugPlural}}", admin.{{.NamePlural}}Index()))
}

// {{.Name}}IndexAction lists {{.SlugPlural}} with server-side search, field
// filters, ordering and pagination. ` + "`format=csv`" + ` streams the matching rows
// as a CSV download instead of JSON.
func {{.Name}}IndexAction(c *gin.Context) {
	cond := {{.Name}}ListCondition(c)

	if c.Query("format") == "csv" {
		items, err := repo.Find[models.{{.Name}}](repo.CurrentDB(),
			sql.Select("*").From("{{.SlugPlural}}").Where(cond).OrderBy("id DESC"))
		if err != nil {
			render.Error(c, err)
			return
		}
		{{.Name}}RenderCSV(c, items)
		return
	}

	page, pageSize := adminListPaging(c)

	total, err := repo.Count(repo.CurrentDB(),
		sql.SelectColumns("count(*)").From("{{.SlugPlural}}").Where(cond))
	if err != nil {
		render.Error(c, err)
		return
	}

	items, err := repo.Find[models.{{.Name}}](repo.CurrentDB(),
		sql.Select("*").From("{{.SlugPlural}}").Where(cond).
			OrderBy(adminListOrder(c, {{.Name}}SortableFields, "id DESC")).
			Limit(pageSize).Offset((page-1)*pageSize))
	if err != nil {
		render.Error(c, err)
		return
	}

	render.OK(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

// {{.Name}}ListCondition builds the WHERE clause: ` + "`q`" + ` searches the text
// fields, every declared column filters exactly, and soft-deleted rows are
// excluded. Identifiers come from the generated whitelist; request input is
// only ever bound as values.
func {{.Name}}ListCondition(c *gin.Context) sql.CondBuilder {
	var conds []sql.CondBuilder

	if q := strings.TrimSpace(c.Query("q")); q != "" {
		like := "%" + q + "%"
		conds = append(conds, &sql.OrCond{Conds: []*sql.Condition{
{{range .Fields}}{{if .Searchable}}			{Key: "{{.JSON}}", Op: "{{$.SearchOp}}", Val: like},
{{end}}{{end}}		}})
	}

{{range .Fields}}{{if and .Form .FilterExpr}}	if v := c.Query("{{.JSON}}"); v != "" {
{{if eq .FilterExpr "v"}}		conds = append(conds, &sql.Condition{Key: "{{.JSON}}", Op: "=", Val: v})
{{else}}		if val, ok := {{.FilterExpr}}; ok {
			conds = append(conds, &sql.Condition{Key: "{{.JSON}}", Op: "=", Val: val})
		}
{{end}}	}
{{end}}{{end}}{{if .SoftDeletable}}	conds = append(conds, &sql.Condition{Key: "deleted_at", Op: "=", Val: nil})
{{end}}	if len(conds) == 0 {
		return nil
	}

	var cond sql.CondBuilder = conds[0]
	for _, next := range conds[1:] {
		cond = &sql.ConditionGroup{Left: cond, Op: sql.And, Right: next}
	}
	return cond
}

// {{.Name}}RenderCSV streams items as a CSV download. encoding/csv quotes
// embedded separators, and adminCSVCell neutralizes leading formula
// characters so spreadsheets do not execute cell content.
func {{.Name}}RenderCSV(c *gin.Context, items []*models.{{.Name}}) {
	c.Header("Content-Disposition", "attachment; filename={{.SlugPlural}}-"+time.Now().Format("20060102150405")+".csv")
	c.Header("Content-Type", "text/csv; charset=utf-8")

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"id", "created_at", "updated_at", {{range .Fields}}{{if .Form}}"{{.DisplayName}}", {{end}}{{end}}})
	for _, row := range items {
		_ = w.Write([]string{
			strconv.FormatInt(int64(row.ID), 10),
			row.CreatedAt.Format(time.RFC3339),
			row.UpdatedAt.Format(time.RFC3339),
{{range .Fields}}{{if .Form}}			{{.CSVExpr}},
{{end}}{{end}}		})
	}
	w.Flush()
}

type {{.Slug}}Params struct {
{{range .Fields}}{{if .Form}}	{{.Name}} {{.GoType}} ` + "`json:\"{{.JSON}}\"`" + `
{{end}}{{end}}}

// {{.Name}}CreateAction inserts a {{.Slug}}.
func {{.Name}}CreateAction(c *gin.Context) {
	var p {{.Slug}}Params
	if err := c.BindJSON(&p); err != nil {
		render.Error(c, err)
		return
	}

	item, err := repo.CreateFrom[models.{{.Name}}](sql.H{
{{range .Fields}}{{if .Form}}			"{{.JSON}}": p.{{.Name}},
{{end}}{{end}}	})
	if err != nil {
		render.Error(c, err)
		return
	}

	adminAudit(c, "create", "{{.SlugPlural}}", int64(item.ID))
	render.OK(c, item)
}

// {{.Name}}UpdateAction modifies a {{.Slug}} by id.
func {{.Name}}UpdateAction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	var p {{.Slug}}Params
	if err := c.BindJSON(&p); err != nil {
		render.Error(c, err)
		return
	}

	{{if .SoftDeletable}}if err := repo.UpdateWhere[models.{{.Name}}](sql.H{
{{range .Fields}}{{if .Form}}			"{{.JSON}}": p.{{.Name}},
{{end}}{{end}}	}, &sql.AndCond{Conds: []*sql.Condition{
		{Key: "id", Op: "=", Val: id},
		{Key: "deleted_at", Op: "=", Val: nil},
	}}); err != nil {{"{"}}
{{else}}	if err := repo.UpdateByID[models.{{.Name}}](sql.IdType(id), sql.H{
{{range .Fields}}{{if .Form}}			"{{.JSON}}": p.{{.Name}},
{{end}}{{end}}	}); err != nil {{"{"}}
{{end}}
		render.Error(c, err)
		return
	}

	adminAudit(c, "update", "{{.SlugPlural}}", id)
	render.Empty(c)
}

// {{.Name}}DestroyAction deletes a {{.Slug}} by id{{if .SoftDeletable}} — a soft
// delete: it stamps deleted_at and leaves the row in place{{end}}.
func {{.Name}}DestroyAction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

{{if .SoftDeletable}}	if err := repo.UpdateWhere[models.{{.Name}}](sql.H{
		"deleted_at": time.Now(),
	}, &sql.AndCond{Conds: []*sql.Condition{
		{Key: "id", Op: "=", Val: id},
		{Key: "deleted_at", Op: "=", Val: nil},
	}}); err != nil {{"{"}}
{{else}}	if err := repo.DeleteByID[models.{{.Name}}](sql.IdType(id)); err != nil {{"{"}}
{{end}}
		render.Error(c, err)
		return
	}

	adminAudit(c, "delete", "{{.SlugPlural}}", id)
	render.Empty(c)
}
`

const adminTypesTemplate = `package admin

import (
	"time"
)

// NavItem is one sidebar entry of the admin layout.
type NavItem struct {
	Label  string
	Href   string
	Active bool
}

// ResourceCard is one dashboard card linking to a resource page.
type ResourceCard struct {
	Label string
	Href  string
	Count int64
}

// AuditLogEntry is one recorded mutating action, shown on the audit page.
type AuditLogEntry struct {
	Time       time.Time
	Username    string
	Action      string
	Resource   string
	ResourceID int64
}
`

const adminLayoutTemplate = `package admin

import (
	"github.com/daqing/airway/app/assets"
	"github.com/daqing/airway/lib/utils"
)

// AdminLayout is the shell for every admin page: a full HTML document with a
// sidebar. The nav entries come from the resource registry in admin_api.
templ AdminLayout(title string, nav []NavItem, content templ.Component) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>{ title } · Admin</title>
			@assets.Stylesheet()
			if utils.AppConfig().IsLocal {
				<script src={ templ.URL(utils.URLPrefix() + "/assets/livereload.js") } data-ws={ utils.URLPrefix() + "/ws" } defer></script>
			}
			@assets.Scripts()
		</head>
		<body class="aw-admin-body">
			<div class="aw-admin-shell">
				<aside class="aw-admin-sidebar">
					<div class="aw-admin-brand">Admin</div>
					<nav class="aw-admin-nav">
						for _, item := range nav {
							<a href={ templ.URL(utils.URLPrefix() + item.Href) } class={ "aw-admin-nav-link", templ.KV("active", item.Active) }>{ item.Label }</a>
						}
					</nav>
					<form method="post" action={ templ.URL(utils.URLPrefix() + "/admin/logout") } class="aw-admin-logout">
						<button type="submit" class="aw-button">Sign out</button>
					</form>
				</aside>
				<main class="aw-admin-main">
					<header class="aw-admin-header">
						<h1>{ title }</h1>
					</header>
					@content
				</main>
			</div>
		</body>
	</html>
}
`

const adminDashboardTemplate = `package admin

import (
	"fmt"

	"github.com/daqing/airway/lib/utils"
)

// Dashboard renders one card per registered admin resource.
templ Dashboard(cards []ResourceCard) {
	<div class="aw-admin-dashboard">
		for _, card := range cards {
			<a class="aw-admin-card" href={ templ.URL(utils.URLPrefix() + card.Href) }>
				<span class="aw-admin-card-count">{ fmt.Sprintf("%d", card.Count) }</span>
				<span class="aw-admin-card-label">{ card.Label }</span>
			</a>
		}
	</div>
}
`

const adminLoginTemplate = `package admin

import (
	"github.com/daqing/airway/app/assets"
	"github.com/daqing/airway/lib/utils"
)

// Login is the standalone admin sign-in page (no layout, no nav). The CSRF
// token arrives both as a cookie and as this hidden field; the POST handler
// requires them to match.
templ Login(errorMessage string, csrfToken string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1"/>
			<title>Admin · Sign in</title>
			@assets.Stylesheet()
			if utils.AppConfig().IsLocal {
				<script src={ templ.URL(utils.URLPrefix() + "/assets/livereload.js") } data-ws={ utils.URLPrefix() + "/ws" } defer></script>
			}
			@assets.Scripts()
		</head>
		<body class="aw-admin-body">
			<div class="aw-admin-login">
				<form method="post" action={ templ.URL(utils.URLPrefix() + "/admin/login") } class="aw-admin-login-card">
					<h1>Admin</h1>
					if errorMessage != "" {
						<p class="aw-field-error" role="alert">{ errorMessage }</p>
					}
					<input type="hidden" name="csrf_token" value={ csrfToken }/>
					<div class="aw-field">
						<label class="aw-field-label" for="username">Username</label>
						<input id="username" name="username" type="text" class="aw-input" required autocomplete="username" autofocus/>
					</div>
					<div class="aw-field">
						<label class="aw-field-label" for="password">Password</label>
						<input id="password" name="password" type="password" class="aw-input" required autocomplete="current-password"/>
					</div>
					<button type="submit" class="aw-button aw-button-primary">Sign in</button>
				</form>
			</div>
		</body>
	</html>
}
`

const adminAuditLogTemplate = `package admin

import (
	"fmt"
)

// AuditLogPage renders the read-only audit trail: who changed what, when.
templ AuditLogPage(entries []AuditLogEntry, page, pageCount int, prefix string) {
	<div class="aw-admin-crud">
		<div class="aw-table-wrap">
			<table class="aw-table">
				<thead>
					<tr>
						<th>Time</th>
						<th>Admin</th>
						<th>Action</th>
						<th>Resource</th>
						<th>Record</th>
					</tr>
				</thead>
				<tbody>
					for _, entry := range entries {
						<tr>
							<td>{ entry.Time.Format("2006-01-02 15:04:05") }</td>
							<td>{ entry.Username }</td>
							<td>{ entry.Action }</td>
							<td>{ entry.Resource }</td>
							<td>{ fmt.Sprintf("%d", entry.ResourceID) }</td>
						</tr>
					}
				</tbody>
			</table>
		</div>
		if pageCount > 1 {
			<div class="aw-pagination">
				if page > 1 {
					<a href={ templ.URL(fmt.Sprintf("%s/admin/audit-log?page=%d", prefix, page-1)) }>← Previous</a>
				}
				<span>{ fmt.Sprintf("Page %d of %d", page, pageCount) }</span>
				if page < pageCount {
					<a href={ templ.URL(fmt.Sprintf("%s/admin/audit-log?page=%d", prefix, page+1)) }>Next →</a>
				}
			</div>
		}
	</div>
}
`

const adminPageTemplate = `package admin

import (
	"github.com/daqing/airway/app/assets"
)

// {{.NamePlural}}Index hosts the {{.Slug}} CRUD island.
templ {{.NamePlural}}Index() {
	<div class="aw-admin-crud">
		@assets.Island("admin-{{.SlugPlural}}-crud", map[string]any{})
	</div>
}
`

const adminIslandTemplate = `import { useCallback, useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import type { ColumnDef } from "@tanstack/react-table";

import {
  apiFetch,
  Button,
  Checkbox,
  DataTable,
  Field,
  Form,
  Input,
  Modal,
  Pagination,
  Select,
  Textarea,
  useToast,
} from "../ui";

const PAGE_SIZE = 20;

type Row = {
  id: number;
{{range .Fields}}  {{.JSON}}: {{.TSRowType}};
{{end}}  created_at?: string;
};

type FormValues = {
{{range .Fields}}  {{.JSON}}: string;
{{end}}};
{{if .HasRefs}}
type RefOption = { id: number; label: string };

function refLabelOf(row: any): string {
  for (const key of ["name", "title", "label", "email"]) {
    if (typeof row[key] === "string" && row[key] !== "") return row[key];
  }
  return "#" + String(row.id);
}

function refLabel(options: RefOption[] | undefined, id: number | null): string {
  if (id == null) return "—";
  const hit = (options ?? []).find((o) => o.id === id);
  return hit ? hit.label : "#" + String(id);
}
{{end}}{{if .HasDatetime}}
function toLocalInput(iso: string | null | undefined): string {
  if (!iso) return "";
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return d.getFullYear() + "-" + pad(d.getMonth() + 1) + "-" + pad(d.getDate()) + "T" + pad(d.getHours()) + ":" + pad(d.getMinutes());
}
{{end}}
export default function {{.NamePlural}}Crud() {
  const toast = useToast();
  const [rows, setRows] = useState<Row[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Row | "new" | null>(null);
{{if .HasRefs}}  const [refOptions, setRefOptions] = useState<Record<string, RefOption[]>>({});
{{end}}{{if .HasAttachment}}  const [uploads, setUploads] = useState<Record<string, string>>({});
{{end}}  const form = useForm<FormValues>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch<{ items: Row[]; total: number }>(
        "/api/v1/admin/{{.SlugPlural}}?page=" + page + "&page_size=" + PAGE_SIZE + "&q=" + encodeURIComponent(search)
      );
      setRows(res.items);
      setTotal(res.total);
    } catch (e) {
      toast.error((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [toast, page, search]);

  useEffect(() => { load(); }, [load]);
{{if .HasRefs}}
  useEffect(() => {
    (async () => {
{{range .RefTargets}}      try {
        const list = await apiFetch<any[]>("/api/v1/admin/{{.Plural}}?page_size=100");
        setRefOptions((prev) => ({ ...prev, {{.Plural}}: list.map((item) => ({ id: item.id, label: refLabelOf(item) })) }));
      } catch {
        // {{.Plural}} list unavailable; the select simply has no options.
      }
{{end}}    })();
  }, []);
{{end}}
  const submitSearch = (e: Event) => {
    e.preventDefault();
    setPage(1);
    setSearch(searchInput.trim());
  };

  const openNew = () => {
    form.reset({{"{"}}{{range $i, $f := .Fields}}{{if $i}}, {{end}}{{$f.JSON}}: ""{{end}}{{"}"}});
{{if .HasAttachment}}    setUploads({});
{{end}}    setEditing("new");
  };

  const openEdit = (row: Row) => {
    form.reset({
{{range .Fields}}      {{.JSON}}: {{.TSPrefill}},
{{end}}    });
{{if .HasAttachment}}    {{.UploadsReset}}
{{end}}    setEditing(row);
  };

  const save = async (values: FormValues) => {
    const payload = {
{{range .Fields}}      {{.JSON}}: {{.TSPayload}},
{{end}}    };
    try {
      if (editing === "new") {
        await apiFetch("/api/v1/admin/{{.SlugPlural}}", { method: "POST", body: JSON.stringify(payload) });
      } else if (editing) {
        await apiFetch("/api/v1/admin/{{.SlugPlural}}/" + editing.id, { method: "PUT", body: JSON.stringify(payload) });
      }
      toast.success(editing === "new" ? "Created" : "Updated");
      setEditing(null);
      await load();
    } catch (e) {
      toast.error((e as Error).message);
    }
  };

  const remove = async (row: Row) => {
    try {
      await apiFetch("/api/v1/admin/{{.SlugPlural}}/" + row.id, { method: "DELETE" });
      toast.show("Deleted #" + row.id, "error");
      await load();
    } catch (e) {
      toast.error((e as Error).message);
    }
  };
{{if .HasAttachment}}
  const uploadFile = async (event: Event, field: string) => {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    const body = new FormData();
    body.append("file", input.files[0]);
    try {
      const res = await apiFetch<{ url: string }>("/api/v1/admin/uploads", { method: "POST", body });
      setUploads((prev) => ({ ...prev, [field]: res.url }));
      form.setValue(field, res.url);
      toast.success("Uploaded");
    } catch (e) {
      toast.error((e as Error).message);
    }
  };
{{end}}
  const columns: ColumnDef<Row, any>[] = [
{{range .Fields}}    {{.TSColumn}}
{{end}}    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <span style="display: inline-flex; gap: 6px;">
          <Button size="sm" onClick={() => openEdit(row.original)}>Edit</Button>
          <Button size="sm" variant="danger" onClick={() => remove(row.original)}>Delete</Button>
        </span>
      ),
    },
  ];

  const editTitle = editing === "new" ? "New {{.Slug}}" : editing ? "Edit #" + editing.id : "";
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div>
      <div class="aw-admin-toolbar">
        <Button variant="primary" onClick={openNew}>New {{.Slug}}</Button>
        <form class="aw-admin-search" onSubmit={submitSearch}>
          <Input
            placeholder="Search…"
            value={searchInput}
            onInput={(e) => setSearchInput((e.target as HTMLInputElement).value)}
          />
          <button type="submit" class="aw-button">Search</button>
          {search !== "" && (
            <button
              type="button"
              class="aw-button aw-button-ghost"
              onClick={() => { setSearchInput(""); setSearch(""); setPage(1); }}
            >
              Clear
            </button>
          )}
        </form>
        <a
          class="aw-button aw-button-secondary"
          href={"/api/v1/admin/{{.SlugPlural}}?format=csv&page_size=100&q=" + encodeURIComponent(search)}
        >
          Export CSV
        </a>
      </div>
      <DataTable columns={columns} data={rows} loading={loading} />
      {pageCount > 1 && <Pagination page={page} pageCount={pageCount} onPageChange={setPage} />}

      <Modal open={editing !== null} onClose={() => setEditing(null)} title={editTitle}>
        <Form
          form={form}
          submitLabel={editing === "new" ? "Create" : "Save"}
          onSubmit={save}
          secondary={<Button onClick={() => setEditing(null)}>Cancel</Button>}
        >
{{range .Fields}}{{.TSControl}}
{{end}}        </Form>
      </Modal>
    </div>
  );
}
`

const adminMigrationUpTemplate = `{{if .Auth}}-- Admin authentication tables.
CREATE TABLE admin_users (
	{{.IDColumn}},
	username VARCHAR(255) NOT NULL,
	password_digest VARCHAR(255) NOT NULL,
	role VARCHAR(20) NOT NULL DEFAULT 'editor',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_admin_users_username ON admin_users (username);

CREATE TABLE admin_sessions (
	{{.IDColumn}},
	token VARCHAR(64) NOT NULL,
	admin_user_id BIGINT NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (admin_user_id) REFERENCES admin_users(id)
);
CREATE INDEX idx_admin_sessions_token ON admin_sessions (token);

CREATE TABLE admin_audit_logs (
	{{.IDColumn}},
	admin_user_id BIGINT NOT NULL DEFAULT 0,
	admin_username VARCHAR(255) NOT NULL DEFAULT '',
	action VARCHAR(20) NOT NULL,
	resource VARCHAR(100) NOT NULL DEFAULT '',
	resource_id BIGINT NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_admin_audit_logs_created_at ON admin_audit_logs (created_at);

{{else if .AuthUpgrade}}-- Upgrade a pre-roles/audit admin install. The legacy email
-- column is kept as-is; sign-in now uses username.
ALTER TABLE admin_users ADD COLUMN username VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE admin_users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'editor';

CREATE TABLE admin_audit_logs (
	{{.IDColumn}},
	admin_user_id BIGINT NOT NULL DEFAULT 0,
	admin_username VARCHAR(255) NOT NULL DEFAULT '',
	action VARCHAR(20) NOT NULL,
	resource VARCHAR(100) NOT NULL DEFAULT '',
	resource_id BIGINT NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_admin_audit_logs_created_at ON admin_audit_logs (created_at);

{{end}}-- Admin resource tables, generated from config/admin.toml.
{{range $table := .Tables}}CREATE TABLE {{$table.SlugPlural}} (
	{{$.IDColumn}},
{{range .Fields}}	{{.SQLCol}},
{{end}}	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP{{range .Fields}}{{if eq .Kind "references"}},
	FOREIGN KEY ({{.JSON}}) REFERENCES {{.RefPlural}}(id){{end}}{{end}}
);
{{range .Fields}}{{if eq .Kind "references"}}CREATE INDEX idx_{{$table.SlugPlural}}_{{.JSON}} ON {{$table.SlugPlural}} ({{.JSON}});
{{end}}{{end}}{{end}}`

const adminMigrationDownTemplate = `{{if or .Auth .AuthUpgrade}}DROP TABLE IF EXISTS admin_audit_logs;
{{end}}{{range .Reversed}}DROP TABLE IF EXISTS {{.SlugPlural}};
{{end}}{{if .Auth}}DROP TABLE IF EXISTS admin_sessions;
DROP TABLE IF EXISTS admin_users;
{{end}}`
