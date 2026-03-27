# Airway CLI Scaffolding Guide

Airway includes a built-in scaffolding CLI. You can use it through the main command:

```bash
go run . cli ...
```

When running `airway cli ...`, Airway automatically tries to load `.env` from the current project root first.

If you already built the binary, the same commands also work as:

```bash
./airway cli ...
```

## Command Overview

```bash
airway cli generate [action|api|model|migration|service|cmd] [params]
airway cli migrate [version]
airway cli rollback [step]
airway cli status
airway cli plugin install /path/to/project
```

## Code Generators

### Generate an API module

```bash
go run . cli generate api admin
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
go run . cli generate action admin show
```

This creates:

- `app/api/admin_api/show_action.go`

Use this when the API folder already exists and you only need a new endpoint handler.

### Generate a model

```bash
go run . cli generate model post
```

This creates:

- `app/models/post.go`

The generated model includes:

- `ID`, `CreatedAt`, `UpdatedAt`
- `TableName()`
- REPL registration via `registerREPLModel`

### Generate a service

```bash
go run . cli generate service post title:string published:bool
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
go run . cli generate cmd post title published
```

This creates:

- `cmd/post.go`

This generator is useful if your project exposes extra custom CLI helpers around generated services.

### Generate a migration

```bash
go run . cli generate migration create_posts
```

This creates a new SQL migration file under:

- `db/migrate`

## Migration Commands

### Run all pending migrations

```bash
go run . cli migrate
```

### Migrate to a specific version

```bash
go run . cli migrate 20260327120000
```

### Roll back the latest migration

```bash
go run . cli rollback
```

### Roll back multiple steps

```bash
go run . cli rollback 3
```

### Show migration status

```bash
go run . cli status
```

Migration commands read:

1. `AIRWAY_DB_DSN`
2. `AIRWAY_PG` as a legacy fallback

In normal local development, these values can come directly from your project's `.env` file because `airway cli ...` loads it automatically.
The migration commands use the current Airway DSN and work with the databases supported by the project, including PostgreSQL, MySQL, and SQLite.

## Plugin Installation

Install the current project as a plugin into another Airway project:

```bash
go run . cli plugin install /path/to/project
```

This copies:

- `./app/*` into `/path/to/project/app/`
- `./cmd/*` into `/path/to/project/cmd/`
- `./db/migrate/*.sql` into `/path/to/project/db/migrate/`

Migration files copied through plugin install are prefixed with a fresh timestamp to avoid version collisions.

## Practical Example

Here is a minimal workflow for adding a `posts` feature from scratch.

### Step 1. Generate the database migration

```bash
go run . cli generate migration create_posts
```

Then edit the generated SQL file in `db/migrate/` and define the table you need.

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
go run . cli migrate
```

### Step 2. Generate the model

```bash
go run . cli generate model post
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
go run . cli generate service post title:string published:bool
```

This creates `app/services/post.go` with basic CRUD helpers.

### Step 4. Generate the API module

```bash
go run . cli generate api post
go run . cli generate action post create
go run . cli generate action post show
```

This gives you:

- `app/api/post_api/routes.go`
- `app/api/post_api/index_action.go`
- `app/api/post_api/create_action.go`
- `app/api/post_api/show_action.go`

### Step 5. Wire the API routes into the router

Open [config/routes.go](/Users/daqing/mzevo/open-source/airway/config/routes.go) and import the generated package:

```go
import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/api/post_api"
	"github.com/daqing/airway/app/api/up_api"
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
go run .
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
