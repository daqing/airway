package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/sql"
	"github.com/daqing/airway/lib/utils"
)

const adminTestConfig = `
[category]
name = "string"
sort_order = "integer"
parent_id = "references:category"

[post]
title = "string"
body = "text"
views = "integer"
score = "float"
published = "boolean"
published_at = "datetime"
status = "enum:draft,published,archived"
cover = "attachment"
category_id = "references"
member_id = "references:member"
deleted_at = "datetime"

[member.meta]
label = "成员"
labels = { name = "姓名", email = "邮箱" }

[member]
name = "string"
email = "string"
`

const adminTestRoutesGo = `package config

import "github.com/gin-gonic/gin"

// PublicRoutes registers the user-facing routes.
func PublicRoutes(r *gin.Engine) {
	plugin.MountAll(r)
}
`

func writeAdminTestProject(t *testing.T) string {
	t.Helper()

	wd := useTempWorkingDir(t)
	makeDirs(t, filepath.Join(wd, "config"))

	writeFile := func(path, content string) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeFile(filepath.Join(wd, "go.mod"), "module github.com/test/app\n\ngo 1.27\n")
	writeFile(filepath.Join(wd, "config", "admin.toml"), adminTestConfig)
	writeFile(filepath.Join(wd, "config", "routes.go"), adminTestRoutesGo)

	return wd
}

func TestParseAdminConfig(t *testing.T) {
	cfg, err := parseAdminConfig([]byte(adminTestConfig))
	if err != nil {
		t.Fatalf("parseAdminConfig: %v", err)
	}

	if len(cfg.Tables) != 3 {
		t.Fatalf("expected 3 tables, got %d", len(cfg.Tables))
	}

	// Tables are slug-alphabetical: category, member, post.
	if cfg.Tables[0].Slug != "category" || cfg.Tables[2].Slug != "post" {
		t.Fatalf("unexpected table order: %s, %s", cfg.Tables[0].Slug, cfg.Tables[2].Slug)
	}

	post := cfg.Tables[2]
	if post.SlugPlural != "posts" || post.NamePlural != "Posts" {
		t.Fatalf("unexpected pluralization: %q / %q", post.SlugPlural, post.NamePlural)
	}

	byJSON := map[string]adminField{}
	for _, field := range post.Fields {
		byJSON[field.JSON] = field
	}

	if f := byJSON["published_at"]; f.Kind != adminKindDatetime || f.GoType != "*time.Time" {
		t.Fatalf("unexpected published_at field: %+v", f)
	}
	if f := byJSON["status"]; f.Kind != adminKindEnum || len(f.Options) != 3 || f.Options[0] != "draft" || f.GoType != "*string" {
		t.Fatalf("unexpected status field: %+v", f)
	}
	if f := byJSON["category_id"]; f.Kind != adminKindReferences || f.RefTarget != "category" || f.RefPlural != "categories" {
		t.Fatalf("unexpected inferred reference: %+v", f)
	}
	if f := byJSON["member_id"]; f.RefTarget != "member" || f.RefPlural != "members" {
		t.Fatalf("unexpected explicit reference: %+v", f)
	}
	if f := byJSON["category_id"]; f.DisplayName != "Category" {
		t.Fatalf("expected reference display name Category, got %q", f.DisplayName)
	}
	if f := byJSON["views"]; f.Kind != adminKindInteger || f.GoType != "int64" {
		t.Fatalf("unexpected views field: %+v", f)
	}

	if !post.HasRefs {
		t.Fatalf("expected post table to flag HasRefs")
	}
	if !post.SoftDeletable {
		t.Fatalf("expected deleted_at to opt post into soft deletes")
	}
	for _, field := range post.Fields {
		if field.JSON == "deleted_at" && field.Form {
			t.Fatalf("deleted_at must be excluded from forms/params/export")
		}
	}

	// [table.meta] overrides the display labels.
	member := cfg.Tables[1]
	if member.Label != "成员" {
		t.Fatalf("expected meta label 成员, got %q", member.Label)
	}
	for _, field := range member.Fields {
		if field.JSON == "name" && field.DisplayName != "姓名" {
			t.Fatalf("expected field label override 姓名, got %q", field.DisplayName)
		}
	}
}

func TestParseAdminConfigErrors(t *testing.T) {
	cases := []struct {
		name   string
		config string
		want   string
	}{
		{
			name:   "unknown field type",
			config: "[post]\ntitle = " + "\"varchar\"\n",
			want:   "unknown type",
		},
		{
			name:   "reserved field",
			config: "[post]\nid = \"string\"\n",
			want:   "reserved",
		},
		{
			name:   "invalid table name",
			config: "[My-Post]\ntitle = \"string\"\n",
			want:   "invalid table name",
		},
		{
			name:   "invalid field name",
			config: "[post]\nTitle = \"string\"\n",
			want:   "invalid field name",
		},
		{
			name:   "empty table",
			config: "[post]\n",
			want:   "no fields",
		},
		{
			name:   "unknown reference target",
			config: "[post]\nuser_id = \"references\"\n",
			want:   "unknown table",
		},
		{
			name:   "enum without options",
			config: "[post]\nstatus = \"enum\"\n",
			want:   "enum requires",
		},
		{
			name:   "enum with empty option",
			config: "[post]\nstatus = \"enum:draft,,published\"\n",
			want:   "must not be empty",
		},
		{
			name:   "duplicate plural",
			config: "[s]\nname = \"string\"\n\n[se]\nname = \"string\"\n",
			want:   "both map to",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseAdminConfig([]byte(tc.config))
			if err == nil {
				// Duplicate plurals only surface via the topo-less parse path;
				// everything else must fail inside parseAdminConfig.
				if tc.want != "both map to" {
					t.Fatalf("expected error containing %q, got nil", tc.want)
				}
				if cfg != nil {
					t.Fatalf("expected nil config for duplicate plural case")
				}
				return
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}
}

func TestParseAdminConfigNonStringField(t *testing.T) {
	_, err := parseAdminConfig([]byte("[post]\nviews = 3\n"))
	if err == nil || !strings.Contains(err.Error(), "invalid admin config") {
		t.Fatalf("expected toml type error, got: %v", err)
	}
}

func TestAdminConfigCircularReferences(t *testing.T) {
	// Cycles surface at topoOrder (parsing only resolves targets).
	cfg, err := parseAdminConfig([]byte("[a]\nb_id = \"references\"\n\n[b]\na_id = \"references\"\n"))
	if err != nil {
		t.Fatalf("parseAdminConfig: %v", err)
	}

	if _, err := cfg.topoOrder(); err == nil || !strings.Contains(err.Error(), "circular") {
		t.Fatalf("expected circular reference error, got: %v", err)
	}
}

func TestAdminConfigTopoOrder(t *testing.T) {
	cfg, err := parseAdminConfig([]byte(adminTestConfig))
	if err != nil {
		t.Fatalf("parseAdminConfig: %v", err)
	}

	ordered, err := cfg.topoOrder()
	if err != nil {
		t.Fatalf("topoOrder: %v", err)
	}

	pos := map[string]int{}
	for i, table := range ordered {
		pos[table.Slug] = i
	}

	// Referenced tables come first; self-references are ignored.
	if pos["category"] > pos["post"] || pos["member"] > pos["post"] {
		t.Fatalf("unexpected order: %v", pos)
	}
}

func TestGenerateAdminEndToEnd(t *testing.T) {
	wd := writeAdminTestProject(t)

	if err := generateAdmin(nil); err != nil {
		t.Fatalf("generateAdmin: %v", err)
	}

	mustContain := func(path string, want ...string) {
		content, err := os.ReadFile(filepath.Join(wd, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, w := range want {
			if !strings.Contains(string(content), w) {
				t.Fatalf("%s: expected %q in:\n%s", path, w, content)
			}
		}
	}

	mustContain(filepath.Join("app", "models", "post.go"),
		"type Post struct",
		"PublishedAt *time.Time",
		"CategoryId *int64",
		`db:"category_id"`,
		"func (Post) Relations()",
		`repo.NewBelongsTo(Category{}, "CategoryId")`,
		`return "posts"`)

	mustContain(filepath.Join("app", "models", "admin_user.go"),
		"type AdminUser struct",
		"Role           string",
		`json:"-"`)

	mustContain(filepath.Join("app", "middlewares", "admin_auth.go"),
		"func AdminAuth(c *gin.Context)",
		"func AdminRequireWrite(c *gin.Context)",
		"func AdminRequireAdmin(c *gin.Context)",
		"admin_session",
		"github.com/test/app/app/models")

	mustContain(filepath.Join("app", "api", "admin_api", "routes.go"),
		"middlewares.AdminAuth",
		"middlewares.AdminRequireWrite",
		"middlewares.AdminRequireAdmin",
		"AuditLogPageAction",
		"resource.Mount(pages, api, writes)")

	mustContain(filepath.Join("app", "api", "admin_api", "registry.go"),
		"func adminListPaging(c *gin.Context)",
		"func adminListOrder(c *gin.Context",
		"func adminAudit(c *gin.Context",
		"func adminCSVCell(v string) string")

	mustContain(filepath.Join("app", "api", "admin_api", "auth_action.go"),
		"ratelimit.New(5, 15*time.Minute)",
		"csrf_token",
		"adminCSRFCookie")

	mustContain(filepath.Join("app", "api", "admin_api", "post_resource.go"),
		"registerAdminResource(AdminResource{",
		"openapi.Get(\"/api/v1/admin/posts\"",
		"type postParams struct",
		"Status *string",
		"repo.CreateFrom[models.Post]",
		"func PostListCondition(c *gin.Context) sql.CondBuilder",
		"adminListPaging(c)",
		"adminListOrder(c, PostSortableFields, \"id DESC\")",
		"format=csv",
		"PostRenderCSV",
		"adminAudit(c, \"create\", \"posts\", int64(item.ID))",
		"adminAudit(c, \"update\", \"posts\", id)",
		"adminAudit(c, \"delete\", \"posts\", id)",
		"Label:  \"Posts\"")

	mustContain(filepath.Join("app", "api", "admin_api", "audit_action.go"),
		"func AuditLogPageAction(c *gin.Context)",
		"admin_audit_logs")

	mustContain(filepath.Join("app", "views", "admin", "admin.templ"),
		"templ AdminLayout(title string, nav []NavItem, content templ.Component)")

	mustContain(filepath.Join("app", "views", "admin", "audit_log.templ"),
		"templ AuditLogPage(entries []AuditLogEntry, page, pageCount int, prefix string)")

	mustContain(filepath.Join("app", "views", "admin", "login.templ"),
		"name=\"csrf_token\"",
		"templ Login(errorMessage string, csrfToken string)")

	mustContain(filepath.Join("app", "views", "admin", "posts_page.templ"),
		`@assets.Island("admin-posts-crud", map[string]any{})`)

	island := filepath.Join("app", "assets", "js", "islands", "admin-posts-crud.tsx")
	mustContain(island,
		"type=\"datetime-local\"",
		"toLocalInput(row.published_at)",
		"refOptions[\"categories\"]",
		"refOptions[\"members\"]",
		"\"/api/v1/admin/uploads\"",
		"value=\"draft\"",
		"Pagination",
		"const PAGE_SIZE = 20",
		"submitSearch",
		"Export CSV",
		"enableSorting: false",
		"page_size=100")

	mustContain(filepath.Join("config", "routes.go"),
		"admin_api.Routes(r)",
		"github.com/test/app/app/api/admin_api")

	// Migration: auth tables first, then referenced tables before referencing
	// ones, with FK constraints and indexes.
	upPath := filepath.Join(wd, "db", "migrate")
	entries, err := os.ReadDir(upPath)
	if err != nil {
		t.Fatalf("read db/migrate: %v", err)
	}
	var upFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			upFile = filepath.Join(upPath, entry.Name())
		}
	}
	if upFile == "" {
		t.Fatalf("no up migration generated in db/migrate")
	}

	up, err := os.ReadFile(upFile)
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	upSQL := string(up)

	for _, want := range []string{
		"CREATE TABLE admin_users",
		"CREATE UNIQUE INDEX idx_admin_users_username",
		"CREATE TABLE admin_sessions",
		"CREATE TABLE categories",
		"CREATE TABLE posts",
		"CHECK (status IN ('draft', 'published', 'archived'))",
		"FOREIGN KEY (category_id) REFERENCES categories(id)",
		"FOREIGN KEY (parent_id) REFERENCES categories(id)",
		"CREATE INDEX idx_posts_category_id ON posts (category_id)",
	} {
		if !strings.Contains(upSQL, want) {
			t.Fatalf("up migration: expected %q in:\n%s", want, upSQL)
		}
	}

	if catPos := strings.Index(upSQL, "CREATE TABLE categories"); catPos == -1 || catPos > strings.Index(upSQL, "CREATE TABLE posts") {
		t.Fatalf("expected categories before posts in:\n%s", upSQL)
	}

	downPath := strings.TrimSuffix(upFile, ".up.sql") + ".down.sql"
	down, err := os.ReadFile(downPath)
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	downSQL := string(down)
	if strings.Index(downSQL, "DROP TABLE IF EXISTS posts") > strings.Index(downSQL, "DROP TABLE IF EXISTS categories") {
		t.Fatalf("expected children dropped first in:\n%s", downSQL)
	}
	if !strings.Contains(downSQL, "DROP TABLE IF EXISTS admin_users") {
		t.Fatalf("expected auth tables dropped in:\n%s", downSQL)
	}
}

func TestGenerateAdminIsAdditive(t *testing.T) {
	writeAdminTestProject(t)

	if err := generateAdmin(nil); err != nil {
		t.Fatalf("first generateAdmin: %v", err)
	}

	migrationsDir := filepath.Join("db", "migrate")
	entries, _ := os.ReadDir(migrationsDir)
	firstRunCount := len(entries)

	// Second run without config changes: everything is skipped, no new files.
	if err := generateAdmin(nil); err != nil {
		t.Fatalf("second generateAdmin: %v", err)
	}

	assertFileCount := func(dir string, want int) {
		t.Helper()
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		if len(entries) != want {
			t.Fatalf("expected %d entries in %s, got %d", want, dir, len(entries))
		}
	}
	assertFileCount(migrationsDir, firstRunCount)

	// Adding a table generates only that table (plus a new migration pair).
	configPath := filepath.Join("config", "admin.toml")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(string(raw)+"\n[tag]\nlabel = \"string\"\n"), 0o644); err != nil {
		t.Fatalf("append config: %v", err)
	}

	if err := generateAdmin(nil); err != nil {
		t.Fatalf("third generateAdmin: %v", err)
	}

	modelContent := readFile(t, filepath.Join("app", "models", "tag.go"))
	if !strings.Contains(modelContent, "type Tag struct") {
		t.Fatalf("expected new tag model, got:\n%s", modelContent)
	}

	entries, _ = os.ReadDir(migrationsDir)
	if len(entries) != firstRunCount+2 {
		t.Fatalf("expected one new migration pair, got %d extra entries", len(entries)-firstRunCount)
	}

	var upFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") && !strings.Contains(readFile(t, filepath.Join(migrationsDir, entry.Name())), "admin_users") {
			upFile = entry.Name()
		}
	}
	if upFile == "" {
		t.Fatalf("expected a second migration without auth tables")
	}
	up := readFile(t, filepath.Join(migrationsDir, upFile))
	if !strings.Contains(up, "CREATE TABLE tags") || strings.Contains(up, "CREATE TABLE posts") {
		t.Fatalf("second migration should only contain the new table:\n%s", up)
	}
}

func TestGenerateAdminMissingConfig(t *testing.T) {
	useTempWorkingDir(t)

	if err := generateAdmin([]string{"config/nope.toml"}); err == nil {
		t.Fatalf("expected error for missing config file")
	}
}

func TestRunAdminUser(t *testing.T) {
	writeAdminTestProject(t)

	// The admin_users table comes from the generated migration; render and
	// apply it against a temp SQLite database.
	dbPath := filepath.Join("tmp", "admin-user-test.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	t.Setenv("AIRWAY_DSN", "sqlite://"+dbPath)

	var upSQL strings.Builder
	tpl, err := template.New("up").Parse(adminMigrationUpTemplate)
	if err != nil {
		t.Fatalf("parse migration template: %v", err)
	}
	if err := tpl.Execute(&upSQL, adminMigrationData{
		IDColumn: "id INTEGER PRIMARY KEY AUTOINCREMENT",
		Auth:     true,
	}); err != nil {
		t.Fatalf("render migration: %v", err)
	}

	db, err := repo.NewDB("sqlite://" + dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	for _, stmt := range strings.Split(upSQL.String(), ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.Conn().Exec(stmt); err != nil {
			t.Fatalf("exec migration statement %q: %v", strings.TrimSpace(stmt), err)
		}
	}

	if err := runAdminRoot([]string{"admin", "s3cret"}); err != nil {
		t.Fatalf("runAdminUser: %v", err)
	}

	if err := runAdminRoot([]string{"admin", "s3cret"}); err == nil {
		t.Fatalf("expected duplicate username error")
	}

	if err := runAdminRoot([]string{"bad name", "s3cret"}); err == nil {
		t.Fatalf("expected invalid username error")
	}

	// admin:root always creates the admin role, even with a --role override.
	if err := runAdminRoot([]string{"admin2", "s3cret", "--role=viewer"}); err == nil {
		t.Fatalf("expected admin:root to reject --role overrides")
	}

	token, err := repo.Count(db, sql.SelectColumns("count(*)").From("admin_users").Where(sql.Eq("username", "admin")))
	if err != nil || token != 1 {
		t.Fatalf("expected exactly one admin user, got %d (err: %v)", token, err)
	}

	row, err := repo.FindOneMap(db, sql.Select("*").From("admin_users").Where(sql.Eq("username", "admin")))
	if err != nil {
		t.Fatalf("find admin user: %v", err)
	}
	digest, _ := row["password_digest"].(string)
	if !utils.ComparePassword(utils.PasswordDigest(digest), "s3cret") {
		t.Fatalf("stored digest does not verify against the given password")
	}
	if role, _ := row["role"].(string); role != "admin" {
		t.Fatalf("expected admin:root to store role admin, got %v", row["role"])
	}
}

func TestGenerateAdminSoftDelete(t *testing.T) {
	wd := writeAdminTestProject(t)

	if err := generateAdmin(nil); err != nil {
		t.Fatalf("generateAdmin: %v", err)
	}

	resource := readFile(t, filepath.Join(wd, "app", "api", "admin_api", "post_resource.go"))
	for _, want := range []string{
		`{Key: "deleted_at", Op: "=", Val: nil}`,
		`"deleted_at": time.Now()`,
		"repo.UpdateWhere[models.Post]",
	} {
		if !strings.Contains(resource, want) {
			t.Fatalf("post_resource.go: expected %q (soft delete) in:\n%s", want, resource)
		}
	}

	// deleted_at is lifecycle, not content: never in params, form controls,
	// columns or the CSV export.
	for _, want := range []string{
		"DeletedAt *time.Time `json:\"deleted_at\"`", // params must not bind it
		"accessorKey: \"deleted_at\"",
		"register(\"deleted_at\")",
	} {
		if strings.Contains(resource, want) {
			t.Fatalf("post_resource.go: unexpected %q", want)
		}
	}

	island := readFile(t, filepath.Join(wd, "app", "assets", "js", "islands", "admin-posts-crud.tsx"))
	if strings.Contains(island, "deleted_at") {
		t.Fatalf("island should not expose deleted_at:\n%s", island)
	}

	// Non-soft-delete resources keep hard deletes.
	member := readFile(t, filepath.Join(wd, "app", "api", "admin_api", "member_resource.go"))
	if !strings.Contains(member, "repo.DeleteByID[models.Member]") {
		t.Fatalf("expected member_resource.go to hard delete:\n%s", member)
	}
}

func TestGenerateAdminForce(t *testing.T) {
	writeAdminTestProject(t)

	if err := generateAdmin(nil); err != nil {
		t.Fatalf("first generateAdmin: %v", err)
	}

	resourcePath := filepath.Join("app", "api", "admin_api", "member_resource.go")
	markerPath := filepath.Join("app", "api", "admin_api", "post_resource.go")

	// Hand edits survive a plain re-run...
	if err := os.WriteFile(markerPath, []byte("// hand edit\n"), 0o644); err != nil {
		t.Fatalf("marker write: %v", err)
	}
	if err := generateAdmin(nil); err != nil {
		t.Fatalf("plain re-run: %v", err)
	}
	if content := readFile(t, markerPath); content != "// hand edit\n" {
		t.Fatalf("plain re-run must not touch existing files, got:\n%s", content)
	}

	// ...but --force=post rewrites only the named table.
	if err := os.WriteFile(resourcePath, []byte("// hand edit\n"), 0o644); err != nil {
		t.Fatalf("marker write: %v", err)
	}

	migrationsDir := filepath.Join("db", "migrate")
	before, _ := os.ReadDir(migrationsDir)

	if err := generateAdmin([]string{"--force=post"}); err != nil {
		t.Fatalf("force run: %v", err)
	}

	if content := readFile(t, markerPath); strings.Contains(content, "hand edit") {
		t.Fatalf("--force should rewrite post_resource.go")
	}
	if content := readFile(t, resourcePath); content != "// hand edit\n" {
		t.Fatalf("--force=post must not touch member_resource.go")
	}

	after, _ := os.ReadDir(migrationsDir)
	if len(after) != len(before) {
		t.Fatalf("--force must not generate migrations")
	}
}

func TestGenerateAdminAuthUpgradeMigration(t *testing.T) {
	wd := writeAdminTestProject(t)

	// Simulate a project generated before roles/audit existed.
	makeDirs(t, filepath.Join(wd, "app", "models"))
	if err := os.WriteFile(filepath.Join(wd, "app", "models", "admin_user.go"), []byte("package models\n"), 0o644); err != nil {
		t.Fatalf("seed admin_user.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wd, "app", "models", "admin_session.go"), []byte("package models\n"), 0o644); err != nil {
		t.Fatalf("seed admin_session.go: %v", err)
	}
	// Some content was already generated (all tables present).
	if err := generateAdmin(nil); err != nil {
		t.Fatalf("generateAdmin: %v", err)
	}

	migrationsDir := filepath.Join(wd, "db", "migrate")
	entries, _ := os.ReadDir(migrationsDir)
	var upgrade string
	for _, entry := range entries {
		content := readFile(t, filepath.Join(migrationsDir, entry.Name()))
		if strings.Contains(content, "ALTER TABLE admin_users ADD COLUMN role") {
			upgrade = content
		}
	}
	if upgrade == "" {
		t.Fatalf("expected an auth upgrade migration with the role column")
	}
	if !strings.Contains(upgrade, "CREATE TABLE admin_audit_logs") {
		t.Fatalf("upgrade migration should create admin_audit_logs:\n%s", upgrade)
	}
}

func TestRunAdminUserRole(t *testing.T) {
	writeAdminTestProject(t)

	dbPath := filepath.Join("tmp", "admin-user-role.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	t.Setenv("AIRWAY_DSN", "sqlite://"+dbPath)

	var upSQL strings.Builder
	tpl, err := template.New("up").Parse(adminMigrationUpTemplate)
	if err != nil {
		t.Fatalf("parse migration template: %v", err)
	}
	if err := tpl.Execute(&upSQL, adminMigrationData{
		IDColumn: "id INTEGER PRIMARY KEY AUTOINCREMENT",
		Auth:     true,
	}); err != nil {
		t.Fatalf("render migration: %v", err)
	}

	db, err := repo.NewDB("sqlite://" + dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	for _, stmt := range strings.Split(upSQL.String(), ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.Conn().Exec(stmt); err != nil {
			t.Fatalf("exec migration statement: %v", err)
		}
	}

	if err := runAdminMember([]string{"viewer", "pw", "--role=viewer"}); err != nil {
		t.Fatalf("runAdminMember viewer: %v", err)
	}
	if err := runAdminMember([]string{"editor", "pw"}); err != nil {
		t.Fatalf("runAdminMember default editor: %v", err)
	}
	if err := runAdminMember([]string{"baduser", "pw", "--role=root"}); err == nil {
		t.Fatalf("expected unknown role error")
	}
	if err := runAdminMember([]string{"superuser", "pw", "--role=admin"}); err == nil {
		t.Fatalf("expected admin:member to reject the admin role")
	}

	row, err := repo.FindOneMap(db, sql.Select("*").From("admin_users").Where(sql.Eq("username", "viewer")))
	if err != nil {
		t.Fatalf("find admin user: %v", err)
	}
	if role, _ := row["role"].(string); role != "viewer" {
		t.Fatalf("expected stored role viewer, got %v", row["role"])
	}

	row, err = repo.FindOneMap(db, sql.Select("*").From("admin_users").Where(sql.Eq("username", "editor")))
	if err != nil {
		t.Fatalf("find editor account: %v", err)
	}
	if role, _ := row["role"].(string); role != "editor" {
		t.Fatalf("expected default role editor, got %v", row["role"])
	}
}
