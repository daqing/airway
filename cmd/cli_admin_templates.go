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
type AdminUser struct {
	ID             airwaysql.IdType ` + "`db:\"id\" json:\"id\"`" + `
	Email          string           ` + "`db:\"email\" json:\"email\"`" + `
	PasswordDigest string           ` + "`db:\"password_digest\" json:\"-\"`" + `
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
	if err != nil {
		adminReject(c)
		return
	}

	if !session.ExpiresAt.After(time.Now()) {
		_ = repo.DeleteByID[models.AdminSession](session.ID)
		adminReject(c)
		return
	}

	user, err := repo.FindByID[models.AdminUser](sql.IdType(session.AdminUserID))
	if err != nil {
		adminReject(c)
		return
	}

	c.Set("admin_user", user)
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
`

const adminRoutesTemplate = `package admin_api

import (
	"github.com/gin-gonic/gin"

	"{{.Module}}/app/middlewares"
)

// Routes mounts the whole admin module: server-rendered pages under /admin
// and the CRUD JSON API under /api/v1/admin. Resource routes and sidebar
// entries register themselves from the per-table *_resource.go init() hooks.
func Routes(r *gin.Engine) {
	r.GET("/admin/login", LoginPageAction)
	r.POST("/admin/login", LoginAction)
	r.POST("/admin/logout", LogoutAction)

	pages := r.Group("/admin", middlewares.AdminAuth)
	{
		pages.GET("", DashboardAction)
	}

	api := r.Group("/api/v1/admin", middlewares.AdminAuth)
	{
		api.POST("/uploads", UploadAction)
	}

	for _, resource := range adminResources {
		resource.Mount(pages, api)
	}
}
`

const adminRegistryTemplate = `package admin_api

import (
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"

	"{{.Module}}/app/views/admin"
	"github.com/daqing/airway/lib/render"
)

// AdminResource describes one generated CRUD resource. Re-running
// ` + "`airway admin:generate`" + ` only adds files, so the registry (and with it the
// sidebar and dashboard) picks up new tables automatically.
type AdminResource struct {
	Plural string
	Label  string
	Count  func() (int64, error)
	Page   func(c *gin.Context)
	Mount  func(pages *gin.RouterGroup, api *gin.RouterGroup)
}

var adminResources []AdminResource

func registerAdminResource(resource AdminResource) {
	adminResources = append(adminResources, resource)
}

// adminNavItems builds the sidebar entries, marking the active page.
func adminNavItems(active string) []admin.NavItem {
	items := []admin.NavItem{{"{{"}}Label: "Dashboard", Href: "/admin", Active: active == ""{{"}}"}}
	for _, resource := range adminResources {
		items = append(items, admin.NavItem{
			Label:  resource.Label,
			Href:   "/admin/" + resource.Plural,
			Active: resource.Plural == active,
		})
	}
	return items
}

// adminPage wraps page content in the admin layout with the sidebar.
func adminPage(title string, active string, content templ.Component) templ.Component {
	return admin.AdminLayout(title, adminNavItems(active), content)
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

	render.HTML(c, adminPage("Dashboard", "", admin.Dashboard(cards)))
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
	"github.com/daqing/airway/lib/render"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"
)

// LoginPageAction serves the sign-in form.
func LoginPageAction(c *gin.Context) {
	render.HTML(c, admin.Login(""))
}

type loginForm struct {
	Email    string ` + "`form:\"email\"`" + `
	Password string ` + "`form:\"password\"`" + `
}

// LoginAction verifies the credentials and starts a cookie session.
func LoginAction(c *gin.Context) {
	var form loginForm
	if err := c.ShouldBind(&form); err != nil {
		render.HTML(c, admin.Login("Invalid email or password"))
		return
	}

	user, err := repo.FindOneBy[models.AdminUser](sql.H{"email": strings.TrimSpace(form.Email)})
	if err != nil || !utils.ComparePassword(utils.PasswordDigest(user.PasswordDigest), form.Password) {
		render.HTML(c, admin.Login("Invalid email or password"))
		return
	}

	token := utils.RandomHex(32)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if _, err := repo.CreateFrom[models.AdminSession](sql.H{
		"token":         token,
		"admin_user_id": user.ID,
		"expires_at":    expiresAt,
	}); err != nil {
		render.HTML(c, admin.Login("Could not sign in: "+err.Error()))
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
				"email":    openapi.Str(),
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
// generator skips files that already exist.
package admin_api

import (
	"strconv"
{{if .HasDatetime}}	"time"
{{end}}	"github.com/gin-gonic/gin"

	"{{.Module}}/app/models"
	"{{.Module}}/app/views/admin"
	"github.com/daqing/airway/lib/openapi"
	"github.com/daqing/airway/lib/render"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
)

func init() {
	registerAdminResource(AdminResource{
		Plural: "{{.SlugPlural}}",
		Label:  "{{.NamePlural}}",
		Count:  func() (int64, error) { return repo.CountEvery[models.{{.Name}}]() },
		Page:   {{.Name}}PageAction,
		Mount:  mount{{.Name}}Routes,
	})

	openapi.Get("/api/v1/admin/{{.SlugPlural}}", func(o *openapi.Operation) {
		o.Summary("Admin: list {{.NamePlural}}").Tag("admin").
			OK(openapi.List[models.{{.Name}}]())
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

func mount{{.Name}}Routes(pages *gin.RouterGroup, api *gin.RouterGroup) {
	pages.GET("/{{.SlugPlural}}", {{.Name}}PageAction)

	api.GET("/{{.SlugPlural}}", {{.Name}}IndexAction)
	api.POST("/{{.SlugPlural}}", {{.Name}}CreateAction)
	api.PUT("/{{.SlugPlural}}/:id", {{.Name}}UpdateAction)
	api.DELETE("/{{.SlugPlural}}/:id", {{.Name}}DestroyAction)
}

// {{.Name}}PageAction renders the admin {{.SlugPlural}} page.
func {{.Name}}PageAction(c *gin.Context) {
	render.HTML(c, adminPage("{{.NamePlural}}", "{{.SlugPlural}}", admin.{{.NamePlural}}Index()))
}

// {{.Name}}IndexAction lists every {{.Slug}} as JSON.
func {{.Name}}IndexAction(c *gin.Context) {
	items, err := repo.FindAll[models.{{.Name}}]()
	if err != nil {
		render.Error(c, err)
		return
	}
	render.OK(c, items)
}

type {{.Slug}}Params struct {
{{range .Fields}}	{{.Name}} {{.GoType}} ` + "`json:\"{{.JSON}}\"`" + `
{{end}}}

// {{.Name}}CreateAction inserts a {{.Slug}}.
func {{.Name}}CreateAction(c *gin.Context) {
	var p {{.Slug}}Params
	if err := c.BindJSON(&p); err != nil {
		render.Error(c, err)
		return
	}

	item, err := repo.CreateFrom[models.{{.Name}}](sql.H{
{{range .Fields}}			"{{.JSON}}": p.{{.Name}},
{{end}}	})
	if err != nil {
		render.Error(c, err)
		return
	}
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

	if err := repo.UpdateByID[models.{{.Name}}](sql.IdType(id), sql.H{
{{range .Fields}}			"{{.JSON}}": p.{{.Name}},
{{end}}	}); err != nil {
		render.Error(c, err)
		return
	}
	render.Empty(c)
}

// {{.Name}}DestroyAction deletes a {{.Slug}} by id.
func {{.Name}}DestroyAction(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		render.ErrorMessage(c, "invalid id")
		return
	}

	if err := repo.DeleteByID[models.{{.Name}}](sql.IdType(id)); err != nil {
		render.Error(c, err)
		return
	}
	render.Empty(c)
}
`

const adminTypesTemplate = `package admin

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

// Login is the standalone admin sign-in page (no layout, no nav).
templ Login(errorMessage string) {
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
					<div class="aw-field">
						<label class="aw-field-label" for="email">Email</label>
						<input id="email" name="email" type="email" class="aw-input" required autocomplete="username"/>
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
  Select,
  Textarea,
  useToast,
} from "../ui";

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
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Row | "new" | null>(null);
{{if .HasRefs}}  const [refOptions, setRefOptions] = useState<Record<string, RefOption[]>>({});
{{end}}{{if .HasAttachment}}  const [uploads, setUploads] = useState<Record<string, string>>({});
{{end}}  const form = useForm<FormValues>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setRows(await apiFetch<Row[]>("/api/v1/admin/{{.SlugPlural}}"));
    } catch (e) {
      toast.error((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);
{{if .HasRefs}}
  useEffect(() => {
    (async () => {
{{range .RefTargets}}      try {
        const list = await apiFetch<any[]>("/api/v1/admin/{{.Plural}}");
        setRefOptions((prev) => ({ ...prev, {{.Plural}}: list.map((item) => ({ id: item.id, label: refLabelOf(item) })) }));
      } catch {
        // {{.Plural}} list unavailable; the select simply has no options.
      }
{{end}}    })();
  }, []);
{{end}}
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

  return (
    <div>
      <div style="margin-bottom: 10px;">
        <Button variant="primary" onClick={openNew}>New {{.Slug}}</Button>
      </div>
      <DataTable columns={columns} data={rows} pageSize={10} loading={loading} />

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
	email VARCHAR(255) NOT NULL,
	password_digest VARCHAR(255) NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_admin_users_email ON admin_users (email);

CREATE TABLE admin_sessions (
	{{.IDColumn}},
	token VARCHAR(64) NOT NULL,
	admin_user_id BIGINT NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (admin_user_id) REFERENCES admin_users(id)
);
CREATE INDEX idx_admin_sessions_token ON admin_sessions (token);

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

const adminMigrationDownTemplate = `{{range .Reversed}}DROP TABLE IF EXISTS {{.SlugPlural}};
{{end}}{{if .Auth}}DROP TABLE IF EXISTS admin_sessions;
DROP TABLE IF EXISTS admin_users;
{{end}}`
