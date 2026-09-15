# Airway CLI Scaffolding Guide

Airway ships a scaffolding CLI. Install it globally with:

```bash
go install github.com/daqing/airway@latest
```

This gives you the `airway` command. Commands auto-load `.env` from the current
project root. Inside a project (or the framework repo itself) the same commands
also work as `go run . <command>` — and some commands (`repl`,
`plugin:install`) *should* be run that way, because they only see the models
and plugins compiled into the running binary (see below).

The legacy form `airway cli <command>` still works as a compatibility alias.

## Command Overview

```bash
airway new <module-path>                                # scaffold a new project skeleton
airway server                                           # start the HTTP server
airway db:create
airway db:drop
airway db:migrate [version]
airway db:rollback [step]
airway db:status
airway plugin:new <module-path>                           # scaffold a new plugin module
airway plugin:list
airway plugin:install <module>
airway generate [action|api|model|migration|service|cmd] [params]
airway schema:dump
airway schema:show
airway upload /path/to/file
airway repl
airway version                                           # or -v / --version; prints the VERSION file contents
```

Running `airway` with no arguments prints usage.

## Start a new project

```bash
airway new myapp                    # directory: myapp
airway new github.com/me/myapp      # module path; directory is the last path segment
```

`airway new` generates a fresh project skeleton based on the framework's `app/`
scaffold, runs `go mod tidy`, and prints the follow-up steps:

```bash
cd myapp
cp .env.example .env    # set DSN and PORT
airway db:create
airway db:migrate
go run .                # starts the server (same as: go run . server)
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

1. `AIRWAY_DB_DSN`
2. `AIRWAY_PG` as a legacy fallback

In normal local development, these values can come directly from your project's `.env` file because the CLI loads it automatically.
The migration commands use the current Airway DSN and work with the databases supported by the project, including PostgreSQL, MySQL, and SQLite.

## Plugin Commands

Scaffold a new plugin module (a standalone Go module; see
[docs/plugin.md](plugin.md)):

```bash
airway plugin:new im                              # directory: im, plugin name: im
airway plugin:new github.com/me/airway-im-plugin  # name derived from the last path segment
```

Unlike the commands below, `plugin:new` works fine with the globally installed
`airway` — it writes files and does not depend on compile-time registration.

Plugins are optional feature modules enabled with blank imports in
`plugins.go` (see [docs/plugin.md](plugin.md)):

```bash
go run . plugin:list           # list registered plugins and mount paths
go run . plugin:install <module> # copy a plugin's embedded SQL migrations into db/migrate
```

Plugins register at compile time, so run these through the project binary
(`go run . ...` in the project directory): the globally installed `airway` can
only list and install the plugins compiled into itself.

`plugin:install` takes the plugin's module path (e.g.
`github.com/daqing/airway-im-plugin`); the plugin name is derived from the
last path segment, same as `plugin:new`. It assigns fresh timestamps to the
copied migrations and skips files that are already installed; afterwards they
are ordinary migrations managed by `db:migrate` / `db:rollback` / `db:status`.

## REPL

```bash
go run . repl
```

The REPL only sees the models compiled into the binary you run — project models
register through `registerREPLModel` in `app/models`, which delegates to
`github.com/daqing/airway/lib/replreg`. Use `go run . repl` inside your project;
the globally installed `airway repl` only sees the framework's built-in models.

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
