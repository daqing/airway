# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project Overview

Airway is a full-stack API framework written in Go, inspired by Ruby on Rails. It is both:

1. A **framework/library** — reusable layers under `lib/` (SQL builder, repository/ORM, migrations, storage, validation, rendering).
2. A **runnable application skeleton** — `main.go` + `app/` + `config/` form a Gin-based HTTP server with WebSocket support, scaffolding CLI, and a REPL.

The project supports PostgreSQL, SQLite 3, and MySQL 8 from the same codebase; the database driver is inferred from the DSN at runtime. Note that some advanced SQL Builder helpers (ARRAY, JSONB, some lateral/window expressions) are still PostgreSQL-oriented.

## Tech Stack

- **Language:** Go 1.26 (module `github.com/daqing/airway`); CGO required (for `mattn/go-sqlite3`).
- **HTTP framework:** Gin (`gin-gonic/gin`), with `gin-contrib/cors`.
- **Database access:** `jmoiron/sqlx` on top of `jackc/pgx/v5` (PostgreSQL), `go-sql-driver/mysql`, `mattn/go-sqlite3`.
- **WebSocket:** `gorilla/websocket`.
- **Cache/queue (optional):** `redis/go-redis/v9`.
- **Cloud storage (optional):** `minio/minio-go/v7` (S3-compatible: S3, Cloudflare R2, Tencent COS).
- **Dev tooling:** `just` (command runner), `overmind` + `air` (process manager + live reload, via `Procfile.dev`).

## Project Layout

```
main.go          Entry point. Dispatches to CLI commands or runs the HTTP server.
app.go           App struct: builds the Gin engine, middleware, routes.
config/          Route registration. config/routes.go wires all app API modules.
app/
  api/           HTTP API modules, one package per namespace (e.g. health_api,
                 home_api, storage_api). Each has routes.go + <name>_action.go.
  models/        Model structs with `db` tags, TableName(), and REPL registration.
  services/      Business-logic layer (currently empty; use `cli generate service`).
  middlewares/   HTTP middlewares (currently empty).
  websocket/     WebSocket hub, connections, publish endpoint.
cmd/             CLI commands: scaffolding generators, db create/drop/migrate/
                 rollback/status, schema dump/show, plugin install, upload, REPL,
                 version.
db/
  migrate/       Migration files live here (imported for side effects).
  schema.json    Schema snapshot, written by `cli schema:dump`.
lib/
  sql/           Dialect-aware SQL builder. Core in lib/sql, dialect overrides in
                 lib/sql/{pg,mysql,sqlite}.
  repo/          Repository/ORM layer: generics-based CRUD (FindBy[User], etc.),
                 Preload (eager loading), Joins, transactions.
  migrate/       Migration internals (dialect compiler, schema state).
  storage/       Unified file storage (local, s3, r2, cos). Access via
                 storage.Current() after boot.
  redis_client/  Redis setup helper.
  render/        JSON response helpers (ok, error, found).
  utils/         Env/config helpers, password hashing, tokens, dates, markdown.
  validation/    Input validation helpers.
docs/            Guides: cli.md, storage.md, docker-compose.yml.example,
                 zh-CN/ (Chinese docs).
data/storage/    Default local file-storage root.
tmp/             Local dev database (airway.db), build artifacts. Git-ignored.
```

## Build, Run, and Test Commands

Prerequisites: copy `.env.example` to `.env` and set `AIRWAY_DB_DSN` and `AIRWAY_PORT`.

```bash
just dev                 # Start dev server with live reload (overmind + air, AIRWAY_ENV=local)
go run .                 # Run the HTTP server directly (requires AIRWAY_ENV and .env)
go build -o ./bin/airway .  # Build binary (CGO_ENABLED=1 needed for sqlite3)
go test ./...            # Run the test suite
go vet ./...             # Lint
```

Other `just` recipes: `just install-deps` (installs air, tmux, overmind), `just docker` (builds the Docker image).

The server requires `AIRWAY_ENV` to be set; when it is `local`, `.env` is loaded automatically. Without arguments the binary starts the HTTP server; with arguments it dispatches to `cmd` (`repl`, `version`, `cli ...`).

## CLI (Scaffolding and Database)

All commands run through the main binary; `cli` commands auto-load `.env` from the project root:

```bash
go run . cli generate api admin                       # new API namespace under app/api/
go run . cli generate action admin show               # new action in an existing API module
go run . cli generate model post                      # new model in app/models/
go run . cli generate service post title:string       # CRUD service in app/services/
go run . cli generate migration create_posts          # new migration in db/migrate/
go run . cli db:create | db:drop
go run . cli db:migrate [version]                     # apply migrations
go run . cli db:rollback [step]
go run . cli db:status
go run . cli schema:dump | schema:show                # writes/reads db/schema.json
go run . cli upload [key] /path/to/file               # upload via configured storage
go run . cli plugin install /path/to/project          # copy app/, cmd/, migrations into another project
go run . repl                                         # interactive repo REPL
```

Migration and schema commands read `AIRWAY_DB_DSN` first and fall back to the legacy `AIRWAY_PG`.

## Code Conventions

- **Models** (`app/models/`): structs with `db:"..."` field tags, a `TableName() string` method, an optional `Relations() map[string]repo.Relation` method for associations, and an `init()` calling `registerREPLModel("Name", Name{})`.
- **API modules** (`app/api/<name>_api/`): one package per namespace. `routes.go` exposes `Routes(r *gin.RouterGroup)`; handlers live in `<action>_action.go` as `func XxxAction(c *gin.Context)`. New modules must be wired into `config/routes.go`.
- **Data access:** prefer the generics API in `lib/repo` (`repo.FindBy[T]`, `repo.CreateFrom[T]`, `repo.UpdateByID[T]`, `repo.DeleteByID[T]`, `repo.Preload(...)`, `repo.Join(...)`) and the `lib/sql` builder for conditions (`sql.Eq`, `sql.And`, `sql.Gt`, ...). Use `repo.Preload` instead of hand-written loops to avoid N+1 queries.
- **Responses:** use the `lib/render` helpers rather than hand-rolled JSON.
- **Storage:** always go through `storage.Current()` — never touch local disk or cloud SDKs directly.
- **Globals at boot:** `main.go` initializes the DB (`repo.SetupDB`), Redis (`redis_client.Setup`), and storage (`storage.Setup`) from environment variables; packages then use their `Current*()` accessors.
- **Naming:** environment variables are prefixed `AIRWAY_`; CLI subcommands follow the Rails-like `db:migrate` / `schema:dump` style.
- Format code with `gofmt`/`go fmt`; keep changes minimal and match the surrounding style.

## Testing

- Standard Go testing: `go test ./...`. Tests live next to the code as `*_test.go` files.
- Most tests are unit tests that run without external services (SQL builder compiles to strings, storage uses a temp local root, etc.).
- **Integration tests are opt-in via environment variables** and skip themselves when unset:
  - `AIRWAY_PG_TEST_DSN` — PostgreSQL integration tests in `lib/repo`.
  - `AIRWAY_MYSQL_TEST_DSN` — MySQL integration tests in `lib/repo`.
- When adding features, add tests in the same package following the existing table-driven style.

## Configuration

All configuration is via environment variables (see `.env.example`):

- `AIRWAY_ENV` — `local` enables `.env` loading and Gin debug mode; anything else runs Gin in release mode.
- `AIRWAY_DB_DSN` — database URL; driver inferred from scheme: `postgres://...`, `sqlite://./tmp/airway.db`, `sqlite://:memory:`, `mysql://...` (native Go MySQL driver DSN format also accepted).
- `AIRWAY_REDIS` — optional Redis URL.
- `AIRWAY_PORT` — listen port (default example: `1900`).
- `STORAGE_DRIVER` — `local` (default), `s3`, `r2`, or `cos`; with `STORAGE_ROOT` for local, or `STORAGE_BUCKET` / `STORAGE_ACCESS_KEY` / `STORAGE_SECRET_KEY` / `STORAGE_REGION` / `STORAGE_ENDPOINT` / optional `STORAGE_PUBLIC_URL` (CDN base; disables presigned URLs) for cloud drivers.
- `TZ` — server timezone (e.g. `Asia/Shanghai`).

## Deployment

- **Docker:** multi-stage `Dockerfile` (golang:1.26-alpine builder with CGO for sqlite3, alpine runtime). Build with `just docker` or `docker build -t airway .`. The image sets `AIRWAY_ENV=production`, exposes port `1900`, and copies `db/` into the image. Note the Dockerfile rewrites apk and Go module proxies to Chinese mirrors (tuna, goproxy.cn).
- **docker-compose:** see `docs/docker-compose.yml.example`.
- The binary is self-contained; run migrations with `./airway cli db:migrate` before/after deploy as needed.

## Security Considerations

- **Never commit `.env`** — it contains database DSNs and storage credentials. Only `.env.example` (with placeholders) is versioned. Do not print secret values in logs or test output.
- The default CORS middleware in `app.go` allows **all origins** with credentials — this is a development default; tighten it before exposing a deployment publicly.
- Always use the `lib/sql` builder / `lib/repo` parameter binding instead of string-concatenating SQL, to avoid SQL injection.
- `repo.DeleteEvery` / full-table writes are intentionally guarded in the REPL (require an explicit `true`); keep similar care in application code.
- The WebSocket publish endpoint (`POST /ws/publish`) and storage upload endpoint (`POST /api/v1/storage`) currently have no authentication — add middleware in `app/middlewares/` before production use.
