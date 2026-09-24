# Airway CLI Scaffolding Guide

Airway ships a scaffolding CLI. Install it globally with:

```bash
go install github.com/daqing/airway@latest
```

This gives you the `airway` command. Commands auto-load `.env` from the current
project root. Inside a project, the globally installed `airway` detects the
host application (go.mod requiring `github.com/daqing/airway` plus a main.go)
and transparently re-runs every project-scoped command through `go run .`, so
plugins, REPL models and Go-code migrations always come from the project's own
binary — one command to remember, `airway <command>`, works everywhere. The
proxy prints a `proxying to project binary: go run . ...` notice on stderr.
The proxy only runs when the globally installed CLI's version matches the
airway version the project pins in go.mod (a `replace` onto a local airway
checkout compares against that checkout's `VERSION` file instead); on a
mismatch the command aborts with an error instead of silently running the
project binary's older CLI logic. `new`, `version` and `help` always run
locally, as does everything outside a project; running `go run . <command>`
yourself remains equivalent.

## Command Overview

```bash
airway new [--local[=path]] <module-path | directory>   # scaffold a new project skeleton
airway server                                           # start the HTTP server
airway db:create
airway db:drop
airway db:migrate [version]
airway db:rollback [step]
airway db:status
airway plugin:new <module-path>                           # scaffold a new plugin module
airway plugin:list
airway plugin:install <module>
airway js:add <pkg>[@version]                          # add a frontend npm dependency (no Node required)
airway js:install                                      # install js.pkg.json deps into app/assets/js/vendor/
airway js:build                                        # bundle the frontend into app/assets/dist
airway generate [action|api|model|migration|service|island|scaffold|cmd] [params]
airway schema:dump
airway schema:show
airway openapi:generate [--out path]                     # write the OpenAPI 3.2 document (default ./openapi.json)
airway templates:compile                                 # regenerate the templ views, then `go generate ./...`
airway admin:generate [config/admin.toml]              # generate the admin backend from a TOML table spec
airway admin:root <username> <password>               # create an administrator (role admin)
airway admin:member <username> <password> [--role=r]  # create a non-admin account (editor|viewer)
airway desktop:init [--force]                           # generate the Wails v3 desktop target in ./desktop (see docs/desktop.md)
airway upload /path/to/file
airway repl
airway version                                           # or -v / --version; prints the VERSION file contents
```

Running `airway` with no arguments prints usage.

## Start a new project

```bash
airway new myapp                    # directory: myapp
airway new github.com/me/myapp      # module path; directory is the last path segment
airway new /path/to/myapp           # create at that local path; module: myapp
airway new sites/myapp              # relative path: same rule (module: myapp)
airway new --local myapp            # develop against the airway checkout in $PWD
airway new --local ~/src/airway myapp
```

`--local` adds a `replace github.com/daqing/airway => <checkout>` to the new
project's go.mod, for developing the framework itself: the scaffolded code can
reference APIs that no published version contains yet. The target must be an
airway source checkout (its go.mod declares the framework module and it has a
`VERSION` file); bare `--local` uses the current directory. Running `new`
from inside the framework repository implies `--local` automatically — the
template comes from that working tree, so the project must depend on it too.

`airway new` generates a fresh project skeleton based on the framework's `app/`
scaffold, seeds `.env` from `.env.example`, runs `go mod tidy`, installs the
frontend dependencies (`airway js:install`), and prints the
follow-up steps:

```bash
cd myapp
# edit .env — set DSN and PORT
airway db:create
airway db:migrate
airway server             # starts the HTTP server
```

A generated project's binary starts the HTTP server when run with no arguments
(or with `server`), and dispatches any other arguments to the built-in CLI.

## Start the server

```bash
airway server        # or, from source: go run . server
```

The framework repository's own `main.go` no longer starts the server by
default — use `go run . server` when developing Airway itself. The Docker image
already runs the binary with `server`.

Environment values always win over `.env` values: `.env` is loaded as a
fallback for keys the process environment does not set, so
`LISTEN=0.0.0.0:1988 go run . server` binds that address even when `.env`
defines `LISTEN`. The value is `host:port` (a bare port is rejected); the
default is `:1900`. The server starts without `AIRWAY_ENV` in the
environment as long as `.env` provides it.

## Upload a file

Upload a local file using the storage configuration from `.env`:

```bash
airway upload /tmp/foo.png
```

The source path becomes a root-relative storage key. In this example the key
is `tmp/foo.png` (shown as `/tmp/foo.png` by the command). The content size is
read from the file and its content type is determined from the extension or
file contents.

To choose the storage key explicitly, pass it before the local file path:

```bash
airway upload images/foo.png /tmp/foo.png
```

## Code Generators

Generators read the module path from the current directory's `go.mod`, so the
generated service/cmd code imports your project's own `app/models` and
`app/services` packages — no hard-coded framework paths.

### Generate an API module

```bash
airway generate api admin
```

This creates:

- `app/api/admin_api/routes.go`
- `app/api/admin_api/index_action.go`

Generated route shape:

```go
func Routes(r *gin.RouterGroup) {
	g := r.Group("/admin")
	{
		g.GET("/index", IndexAction)
	}
}
```

Use this when you want to create a new API namespace quickly.

### Generate an action inside an existing API module

```bash
airway generate action admin show
```

This creates:

- `app/api/admin_api/show_action.go`

Use this when the API folder already exists and you only need a new endpoint handler.

### Generate a model

```bash
airway generate model post
```

This creates:

- `app/models/post.go`

The generated model includes:

- `ID`, `CreatedAt`, `UpdatedAt`
- `TableName()`
- REPL registration via `registerREPLModel`

### Generate a service

```bash
airway generate service post title:string published:bool
```

This creates:

- `app/services/post.go`

The generated file includes:

- `FindPost`
- `CreatePost`
- `UpdatePost`
- `DeletePost`

Field arguments use `name:type` format.

### Generate a command helper

```bash
airway generate cmd post title published
```

This creates:

- `cmd/post.go`

This generator is useful if your project exposes extra custom CLI helpers around generated services.

### Generate a migration

```bash
airway generate migration create_posts
```

This creates a pair of timestamped SQL files under `db/migrate/`:

- `<timestamp>_create_posts.up.sql` — the forward migration
- `<timestamp>_create_posts.down.sql` — the rollback migration

Both files contain commented-out `CREATE TABLE` / `DROP TABLE` examples to get
you started; edit them to define your real schema.

The older Go DSL migration mechanism (`schema.RegisterChange` in
`lib/migrate/schema`) is still supported, but DSL migrations only take effect
when they are compiled into the binary that runs the migration. When the CLI
finds timestamp-named `.go` migration files under `./db/migrate`, it prints a
warning to remind you of this.

## Migration Commands

### Run all pending migrations

```bash
airway db:migrate
```

### Migrate to a specific version

```bash
airway db:migrate 20260327120000
```

### Roll back the latest migration

```bash
airway db:rollback
```

### Roll back multiple steps

```bash
airway db:rollback 3
```

### Show migration status

```bash
airway db:status
```

Migration commands read:

1. `AIRWAY_DSN`
2. `DSN`

In normal local development, these values can come directly from your project's `.env` file because the CLI loads it automatically.
The migration commands use the current Airway DSN and work with the databases supported by the project, including PostgreSQL, MySQL, and SQLite.

## Plugin Commands

Scaffold a new plugin module (a standalone Go module; see
[docs/plugin.md](plugin.md)):

```bash
airway plugin:new im                              # directory: im, plugin name: im
airway plugin:new github.com/me/airway-im-plugin  # name derived from the last path segment
```

Unlike `plugin:list`, `plugin:new` never touches the project binary — it just
writes files.

Plugins are optional feature modules enabled with blank imports in
`plugins.go` (see [docs/plugin.md](plugin.md)):

```bash
airway plugin:list               # list registered plugins and mount paths
airway plugin:install <module>   # install a plugin's SQL migrations, host/ tree, and deps/ directory
```

Plugins register at compile time, so inside a project the globally installed
`airway` re-runs these through `go run .` automatically (the project's binary
is what can see your enabled plugins); `go run . plugin:list` does the same
thing directly.

`plugin:install` takes the plugin's module path (e.g.
`github.com/daqing/airway-im-plugin`); the plugin name is derived from the
last path segment, same as `plugin:new`. It assigns fresh timestamps to the
copied migrations and skips files that are already installed; afterwards they
are ordinary migrations managed by `db:migrate` / `db:rollback` / `db:status`.

## Frontend Dependency Commands

Frontend npm dependencies are managed without a Node toolchain: the CLI
talks to an npm-compatible registry directly and unpacks packages into
`app/assets/js/vendor/` (node_modules-compatible layout for esbuild).

```bash
airway js:add preact                          # latest version, pinned exactly
airway js:add @tanstack/react-table@9.2.4     # explicit version
airway js:install                             # install per js.pkg.json (idempotent)
airway js:build                               # bundle app/assets/js/app.tsx into app/assets/dist
```

With `AIRWAY_ENV=local`, `airway server` additionally builds the bundle in
memory: `/assets/*` serves the freshest build with ETag revalidation, and
editing any `.ts/.tsx/.css` file under `app/assets/js` (vendor/ excluded)
triggers an incremental rebuild plus a livereload page refresh via the app's
WebSocket. `just build` runs `js:build` before compiling the binary; the
`dist/` output is committed, so deploying never requires the bundler.

`js.pkg.json` at the project root has two sections: `deps` (direct
dependencies, exact versions — hand-editable) and `lock` (the fully resolved
tree including transitive dependencies, each with a sha512 integrity; written
by `js:add` / `js:install`). `js:install` installs exactly the locked
versions and re-resolves only when the lock is missing or stale; already
installed packages are skipped.

The registry defaults to `https://registry.npmjs.org`; set
`AIRWAY_JS_REGISTRY` (or `JS_REGISTRY`) to use a mirror, e.g.
`https://registry.npmmirror.com`. The `vendor/` directory is committed so a
fresh clone builds offline.

## OpenAPI Commands

```bash
airway openapi:generate                    # write ./openapi.json (OpenAPI 3.2)
airway openapi:generate --out docs/api.json
```

Writes a deterministic OpenAPI 3.2 document for every route registered on the
Gin engine (plugin routes included); the output is sorted, so regenerating
after a change keeps diffs clean. The file is a local build artifact
(git-ignored). A running server also serves the document live at
`GET /openapi.json` (under the `URL_PREFIX` when one is configured), with a
`servers` entry derived from the request host.

Routes without declarations are documented with the framework's default JSON
envelope; modules add request/response schemas in an `openapi.go` file — see
the [OpenAPI guide](openapi.md) for the declaration API and client-generation
recipes (openapi-typescript / orval for Vue 3 and React,
swift-openapi-generator for SwiftUI).

## REPL

```bash
go run . repl
```

The REPL only sees the models compiled into the binary you run — project models
register through `registerREPLModel` in `app/models`, which delegates to
`github.com/daqing/airway/lib/replreg`. Inside a project the globally installed
`airway repl` proxies to `go run . repl` for you; outside a project it only
sees the framework's built-in models.

## Practical Example

Here is a minimal workflow for adding a `posts` feature from scratch.

### Step 1. Generate the database migration

```bash
airway generate migration create_posts
```

Then edit the generated `.up.sql` file in `db/migrate/` and define the table you need.

Example:

```sql
CREATE TABLE posts (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL,
  published BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Run the migration:

```bash
airway db:migrate
```

### Step 2. Generate the model

```bash
airway generate model post
```

This creates `app/models/post.go`.

At this point you will usually extend the generated struct with your real fields, for example:

```go
type Post struct {
	ID        sql.IdType `db:"id" json:"id"`
	Title     string     `db:"title" json:"title"`
	Published bool       `db:"published" json:"published"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}
```

### Step 3. Generate the service

```bash
airway generate service post title:string published:bool
```

This creates `app/services/post.go` with basic CRUD helpers.

### Step 4. Generate the API module

```bash
airway generate api post
airway generate action post create
airway generate action post show
```

This gives you:

- `app/api/post_api/routes.go`
- `app/api/post_api/index_action.go`
- `app/api/post_api/create_action.go`
- `app/api/post_api/show_action.go`

### Step 5. Wire the API routes into the router

Open [config/routes.go](https://github.com/daqing/airway/blob/main/config/routes.go) and import the generated package:

```go
import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/api/post_api"
	"github.com/daqing/airway/app/api/health_api"
	"github.com/daqing/airway/app/websocket"
)
```

Then register it inside `apiGroupRoutes`:

```go
func apiGroupRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		post_api.Routes(v1)
	}
}
```

With the generated default route file, you will get an endpoint like:

```text
GET /api/v1/post/index
```

### Step 6. Fill in the generated action logic

For example, `create_action.go` is only a scaffold. You still need to:

- define request params
- call `services.CreatePost(...)`
- return data through `render.OK(...)` or `render.Error(...)`

### Step 7. Run the app

```bash
just
```

Or:

```bash
go run . server
```

At that point you have the full skeleton for:

- database migration
- model
- service
- API handlers
- route registration

## Notes

- Generators do not overwrite existing files. If the target file already exists, the command returns `file already exists`.
- `generate api` creates files, but you still need to wire the generated `Routes(...)` into your router setup.
- `generate service` assumes your project has an `app/services` package.
- Generated files are starting points. They are meant to be edited after creation.

## Scaffold a CRUD resource

`airway generate scaffold post title:string` produces the whole vertical
slice — model with fields, a dialect-aware migration (the auto-increment
primary key follows your configured DSN), the CRUD service, a JSON API
under `/api/v1/posts`, a templ page at `/posts`, a server-rendered detail
page at `/posts/:id`, a CRUD island (DataTable + modal form, wired to the
API through `apiFetch`), and a static export registration (`export_posts.go`:
the list page plus one detail page per row; see
[docs/static-export.md](static-export.md)). It also registers the routes in
`config/routes.go`. Afterwards run:

```bash
airway templates:compile   # compile the .templ views (works even on a fresh scaffold)
airway js:build            # bundle the new island
airway db:migrate          # create the table
airway server              # visit /posts
airway static:build        # export the pages + detail pages as static HTML (needs DSN)
```

`airway generate island chart` scaffolds a single interactive island under
`app/assets/js/islands/`; embed it with `@assets.Island("chart", props)`.

## Generate an admin panel

`airway admin:generate` turns a single TOML file into a complete admin
backend: sign-in with cookie sessions, a dashboard with one card per
resource, sidebar navigation, and full CRUD (JSON API + type-aware CRUD
island) for every table. Config lives in `config/admin.toml` by default
(an alternate path can be passed as the only argument). Each top-level
table is one resource, keyed by its **singular** name; each field maps a
column name to a type:

```toml
[category]
name = "string"
sort_order = "integer"
parent_id = "references:category"   # self-reference (explicit target)

[post]
title = "string"
body = "text"
views = "integer"
score = "float"
published = "boolean"
published_at = "datetime"
status = "enum:draft,published,archived"
cover = "attachment"
category_id = "references"          # target inferred from the _id suffix
```

Supported field types: `string`, `text`, `integer`, `float`, `boolean`,
`datetime`, `enum:a,b,c`, `references[:table]` and `attachment`. `id`,
`created_at` and `updated_at` are added to every table automatically and
must not be declared.

Run the generator, then the standard follow-ups:

```bash
airway admin:generate
airway templates:compile                  # compile the .templ views
airway js:build                           # bundle the CRUD islands
airway db:migrate                         # create the tables
airway admin:root admin      # create the first administrator account
airway server                             # visit /admin
```

What gets generated:

- One model per table in `app/models/` (tables with references also get
  `Relations()`), plus `admin_user.go` / `admin_session.go` for auth.
- `app/api/admin_api/`: a self-registering resource file per table with
  CRUD actions and OpenAPI declarations, plus shared files (registry,
  routes, login/logout, attachment uploads through `storage.Current()`).
- `app/views/admin/`: the admin layout with sidebar, dashboard, login
  page, and one page per table hosting its CRUD island.
- One migration pair per run (auth tables on the first run, then new
  tables in foreign-key dependency order).

Routes mount under `/admin` (pages) and `/api/v1/admin` (JSON API). Both
sit behind the `AdminAuth` middleware: pages redirect to `/admin/login`
when there is no valid session, API calls get a 401. Administrators are
created with `airway admin:root <username> <password>`, non-admin accounts
(editor/viewer) with `airway admin:member` — passwords are
bcrypt-hashed and sessions are stored server-side in `admin_sessions`.

Roles and hardening are built in:

- **Roles** — `admin` (full access, sees the audit log), `editor` (default;
  read/write) and `viewer` (read-only: writes and uploads get a 403).
  Mutating routes sit behind `AdminRequireWrite`.
- **CSRF** — the login form carries a double-submit token that must match
  the `airway_admin_csrf` cookie.
- **Rate limiting** — five failed sign-ins for the same IP and username lock
  the pair out for fifteen minutes (`lib/ratelimit`).
- **Audit log** — every create, update and delete is recorded with the
  acting account; admins can review it at `/admin/audit-log`.
- **Server-side lists** — the list API supports `page`, `page_size`,
  `q` (text search), `sort`/`order` (whitelisted columns) and exact-match
  filters on any declared field (`?status=published`). The CRUD island
  ships a search box, pagination and an Export CSV button.
- **Soft delete** — declaring `deleted_at = "datetime"` on a table makes
  destroy stamp the column instead of deleting; reads filter it out.
- **Display labels** — an optional `[table.meta]` section overrides the
  sidebar label (`label`) and per-field labels (`labels`), so the panel can
  speak any language the TOML does.

Re-running `admin:generate` after adding tables to the TOML is additive:
existing files are never rewritten, and the registry picks up new
resources automatically (sidebar and dashboard included). To regenerate an
existing table's code — after editing it in the TOML, or to pick up new
generator features — pass `--force` (or `--force=table1,table2`): it
rewrites that table's generated files and discards hand edits, but never
touches migrations, so schema changes still need a hand-written migration.
Removing a table from the TOML does not delete generated code.
