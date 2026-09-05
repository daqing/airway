# Airway

A full-stack API framework written in Go, inspired by Ruby on Rails. It runs the
same application code against **PostgreSQL**, **MySQL 8** and **SQLite** — the
database driver is inferred from the DSN at runtime.

- **[中文文档](docs/zh-CN/README.md)**
- **[CLI 脚手架指南](docs/cli.md)** / **[文件存储指南](docs/storage.md)**
- **[SQL Builder DSL 指南（中文）](docs/zh-CN/sql-builder.md)**

## What Airway is

Airway is both a **framework/library** and a **runnable application skeleton**:

- Reusable layers under `lib/` — SQL builder, repository/ORM, migrations,
  storage, rendering, validation.
- A Gin-based HTTP server (`main.go` + `app/` + `config/`) with a scaffolding
  CLI, WebSocket support and a REPL, ready to `go run .`.

## Features

- **Generics-based repository** (`lib/repo`): typed `FindBy[User]`,
  `CreateFrom[User]`, preload/eager-loading, joins, transactions.
- **Dialect-aware SQL builder** (`lib/sql` + `pg` / `mysql` / `sqlite`
  dialects); conditions like `sql.Eq`, `sql.AllOf`, `sql.Gt`.
- **Schema-driven migrations**: generate and apply migrations with the CLI;
  SQLite schema changes are handled by table rebuild.
- **Unified file storage** (`lib/storage`): local directory, Amazon S3,
  Cloudflare R2 or Tencent COS — selected purely by configuration, exposed via
  an HTTP upload/download API.
- **Gin web server + WebSocket** pub/sub.
- **HTML views with [templ](https://templ.guide/)**: pages as `.templ`
  templates under `app/views/`, rendered from actions via `lib/render.HTML`.
- **Scaffolding CLI** (`airway cli generate ...`, `db:migrate`, ...).
- **Repo REPL** with typed scan and Go-expression evaluation.
- **Optional sub-path prefix** (`URL_PREFIX`) for deploying behind a reverse
  proxy, e.g. `http://host:1900/airway/...`.

## Quick start

### 1. Get the code and configure `.env`

```bash
git clone https://github.com/daqing/airway.git
cd airway
cp .env.example .env
```

Open `.env` and set at least the database DSN and port:

```env
DSN="sqlite://./tmp/airway.db"     # or postgres://..., mysql://...
PORT="1900"
```

Environment values may use the short name (`DSN`, `PORT`, `REDIS`, `URL_PREFIX`)
or the `AIRWAY_`-prefixed form (`AIRWAY_DSN`, `AIRWAY_PORT`, ...). When both are
set, the `AIRWAY_` form wins. See [Configuration](#configuration).

### 2. Run the server

```bash
go run .   # requires AIRWAY_ENV (e.g. AIRWAY_ENV=local) and a configured DSN
```

Or start the local dev server with live reload:

```bash
just dev
```

The app listens on `http://127.0.0.1:1900` (`GET /` serves an HTML page
rendered with templ, `GET /health` returns `UP`).

## Configuration

All configuration is via environment variables (see `.env.example`). Each value
accepts a short name or its `AIRWAY_` alias, with the alias taking precedence.

| Variable | Description |
| --- | --- |
| `DSN` / `AIRWAY_DSN` | Database URL. Driver is inferred from the scheme (see below). |
| `PORT` / `AIRWAY_PORT` | HTTP listen port (default `1900`). |
| `REDIS` / `AIRWAY_REDIS` | Optional Redis URL for cache/queue. |
| `URL_PREFIX` / `AIRWAY_URL_PREFIX` | Optional public sub-path prefix, e.g. `/airway`. Empty serves at the root. |
| `AIRWAY_ENV` | `local` loads `.env` and uses Gin debug mode; anything else runs release mode. |
| `STORAGE_DRIVER` | `local` (default), `s3`, `r2` or `cos`. |
| `STORAGE_ROOT` | Local storage root (default `./data/storage`). |
| `STORAGE_*` | Cloud settings: `STORAGE_REGION`, `STORAGE_ENDPOINT`, `STORAGE_BUCKET`, `STORAGE_ACCESS_KEY`, `STORAGE_SECRET_KEY`, `STORAGE_PUBLIC_URL` (optional CDN base). |
| `TZ` | Server timezone, e.g. `Asia/Shanghai`. |

### Database DSNs

The same code switches databases by changing only `DSN`:

```env
# PostgreSQL
DSN="postgres://daqing:passwd@127.0.0.1:5432/airway"

# SQLite (file or in-memory) — pure-Go driver, no CGO required
DSN="sqlite://./tmp/airway.db"
DSN="sqlite://:memory:"

# MySQL 8 (URL or native driver format)
DSN="mysql://root:passwd@127.0.0.1:3306/airway?charset=utf8mb4"
DSN="root:passwd@tcp(127.0.0.1:3306)/airway?charset=utf8mb4&parseTime=true"
```

Notes:

- Basic CRUD flows are portable across PostgreSQL, MySQL 8 and SQLite.
- Some advanced SQL Builder helpers (ARRAY, JSONB, a few lateral/window
  expressions) are still PostgreSQL-oriented.
- SQLite support uses the pure-Go `modernc.org/sqlite` driver.

## HTTP endpoints

Registered in `config/routes.go`:

| Method | Path | Description |
| --- | --- | --- |
| GET | `/` | Home page (HTML, rendered from a templ view). |
| GET | `/health` | Health check. |
| GET | `/ws` | WebSocket connection. |
| POST | `/ws/publish` | Publish a message to connected clients (form field `message`). |
| POST | `/api/v1/storage` | Upload a file (multipart `file`, optional `dir`). |
| GET | `/api/v1/storage/*key` | Download a file. |
| DELETE | `/api/v1/storage/*key` | Delete a file. |

When `URL_PREFIX` is set, all of these are also served under the prefix. For
example with `URL_PREFIX="/airway"`:

```bash
curl http://127.0.0.1:1900/airway/health
curl -F "file=@report.pdf" http://127.0.0.1:1900/airway/api/v1/storage
```

Clients — including WebSocket connections — must include the prefix
(`ws://host:1900/airway/ws`). Local storage URLs returned by the API are
prefixed accordingly; cloud/CDN URLs are untouched.

## HTML views (templ)

Pages are [templ](https://templ.guide/) templates under `app/views/`, one
folder per API module — the folder name drops the `_api` suffix, so `home_api`
renders `app/views/home/index.templ`. Each folder is its own package; a shared
document shell lives in `app/views/layouts/base.templ`. Actions serve a
component with the `lib/render` HTML helper:

```go
// app/api/home_api/index_action.go
render.HTML(c, home.Index())
```

After editing any `.templ` file, regenerate the Go code and keep it committed:

```bash
go generate ./...   # or: just generate
```

The generated `*_templ.go` files are committed, so building and testing never
require the templ CLI.

## CLI

All commands run through the main binary; `cli` commands auto-load `.env` from
the project root:

```bash
airway cli generate api admin                  # new API namespace under app/api/
airway cli generate action admin show          # new action in an existing API module
airway cli generate model post                 # new model in app/models/
airway cli generate service post title:string  # CRUD service in app/services/
airway cli generate migration create_posts     # new migration in db/migrate/
airway cli db:create | db:drop
airway cli db:migrate [version]                # apply migrations
airway cli db:rollback [step]
airway cli db:status
airway cli schema:dump | schema:show           # writes / reads db/schema.json
airway cli upload [key] /path/to/file          # upload via the configured storage
airway cli plugin install /path/to/project     # copy app/, cmd/ and migrations into another project
```

Database commands read `DSN`/`AIRWAY_DSN`; the legacy `AIRWAY_DB_DSN` and
`AIRWAY_PG` are still honored for backward compatibility. See the full
[CLI guide](docs/cli.md).

## Repository API (`lib/repo`)

### Models

A model is a struct with `db` tags and a `TableName()` method. Associations are
declared via `Relations()`.

```go
import "github.com/daqing/airway/lib/repo"

type User struct {
	ID      int64    `db:"id"`
	Name    string   `db:"name"`
	Email   string   `db:"email"`
	Profile *Profile // belongs_to
	Posts   []*Post  // has_many
}

func (User) TableName() string { return "users" }

func (User) Relations() map[string]repo.Relation {
	return map[string]repo.Relation{
		"Profile": repo.HasOne(Profile{}, "UserID"),
		"Posts":   repo.HasMany(Post{}, "UserID"),
	}
}

type Post struct {
	ID     int64  `db:"id"`
	UserID int64  `db:"user_id"`
	Title  string `db:"title"`
	Author *User  // belongs_to
}

func (Post) TableName() string { return "posts" }

func (Post) Relations() map[string]repo.Relation {
	return map[string]repo.Relation{
		"Author": repo.NewBelongsTo(User{}, "UserID"),
	}
}
```

### CRUD

All helpers use the DB configured at boot (`repo.SetupDB`) and take the model
type as a type parameter:

```go
// Create
user, err := repo.CreateFrom[User](sql.H{"name": "John", "email": "john@example.com"})

// Read
user, err := repo.FindByID[User](1)
user, err := repo.FindOneBy[User](sql.H{"email": "john@example.com"})
users, err := repo.FindBy[User](sql.H{"active": true})
users, err := repo.FindAll[User]()

// Update
err := repo.UpdateByID[User](1, sql.H{"name": "Jane"})
err := repo.UpdateWhere[User](sql.H{"status": "inactive"}, sql.Eq("last_login_at", nil))

// Delete
err := repo.DeleteByID[User](1)
err := repo.DeleteWhere[User](sql.H{"status": "banned"})

// Exists / Count
ok, err := repo.ExistsWhere[User](sql.H{"email": "john@example.com"})
n, err := repo.CountWhere[User](sql.H{"active": true})
n, err := repo.CountEvery[User]()
```

### Preload (eager loading)

Preload replaces N+1 loops with a couple of queries:

```go
users, _ := repo.FindBy[User](sql.H{})
err := repo.Preload("Profile", "Posts").Exec(&users)

// Nested and conditional
err := repo.Preload("Posts").ThenPreload("Comments").Exec(&users)
err := repo.PreloadCond("Posts", sql.AllOf(
	sql.Eq("published", true),
	sql.Gte("created_at", "2024-01-01"),
)).Exec(&users)
```

### Joins

```go
results, err := repo.Join(User{}).LeftJoins("Profile").Find()
results, err := repo.Join(User{}).
	Joins("Posts", sql.Gt("posts.views", 100)).
	Where(sql.Eq("users.active", true)).
	OrderBy("users.name ASC").
	Page(1, 20).
	Find()

count, err := repo.Join(User{}).Joins("Posts").Count()

var users []*User
err := repo.Join(User{}).LeftJoins("Profile").FindInto(&users)
```

### Rails mapping

| Rails ActiveRecord | Airway |
| --- | --- |
| `User.find(id)` | `repo.FindByID[User](id)` |
| `User.find_by(email: e)` | `repo.FindOneBy[User](sql.H{"email": e})` |
| `User.where(active: true)` | `repo.FindBy[User](sql.H{"active": true})` |
| `User.all` | `repo.FindAll[User]()` |
| `User.create(attrs)` | `repo.CreateFrom[User](attrs)` |
| `User.update(id, attrs)` | `repo.UpdateByID[User](id, attrs)` |
| `User.delete(id)` | `repo.DeleteByID[User](id)` |
| `User.joins(:profile)` | `repo.Join(User{}).Joins("Profile")` |
| `User.includes(:posts)` | `repo.Preload("Posts").Exec(&users)` |
| `User.count` | `repo.CountEvery[User]()` |
| `User.where(active: true).count` | `repo.CountWhere[User](sql.H{"active": true})` |

## Repo REPL

Exercise `lib/repo` directly against the configured database:

```bash
go run . repl                              # uses the configured DSN
go run . repl --driver sqlite --dsn ./tmp/airway.db
```

Commands: `help`, `driver`, `tables`, `exit`. Type a Go expression to evaluate
it — builders print the compiled SQL, `repo.*` calls run against the database:

```text
repo.FindOne("users", sql.Eq("id", 1))
repo.Find[models.User](pg.Select("id").Where(sql.Eq("id", 1)))
repo.Insert[models.User](pg.H{"id": 1234, "email": "dev@example.com"})
repo.Update("users", pg.H{"enabled": false}, sql.Eq("id", 1))
repo.Delete("users", sql.Eq("id", 1))
pg.Select("*").From("users").Where(sql.Eq("id", 1))
```

Available namespaces: `repo`, `sql`, `pg`, `mysql`, `sqlite`, `models`.
`repo.Find`/`FindOne`/`Count`/`Exists` accept a built statement or a
`table + condition`; typed calls support anonymous structs and app models.
Full-table updates/deletes require an explicit `true` as the last argument.

## File storage

`lib/storage` is a unified layer over a local directory or a cloud backend
(S3 / R2 / COS). Configure the backend with `STORAGE_DRIVER`:

```env
# Local
STORAGE_DRIVER="local"
STORAGE_ROOT="./data/storage"

# Cloud (S3-compatible): choose s3 / r2 / cos
STORAGE_DRIVER="s3"
STORAGE_REGION="us-east-1"
STORAGE_BUCKET="my-bucket"
STORAGE_ACCESS_KEY="..."
STORAGE_SECRET_KEY="..."
# STORAGE_PUBLIC_URL="https://cdn.example.com"   # optional CDN base
```

Use it in code via the current backend:

```go
store := storage.Current() // installed at boot by storage.Setup

err := store.Put(ctx, "docs/report.pdf", storage.Object{
	Reader:      file,
	Size:        size,
	ContentType: "application/pdf",
})
rc, err := store.Get(ctx, "docs/report.pdf") // close when done
ok, err := store.Exists(ctx, "docs/report.pdf")
url, err := store.URL(ctx, "docs/report.pdf", 24*time.Hour)
err := store.Delete(ctx, "docs/report.pdf")
```

`URL()` returns the app's own download path for the local backend and a CDN or
presigned URL for cloud backends. The app exposes a REST API over the same
layer:

```bash
# Upload (multipart field "file", optional "dir")
curl -F "file=@report.pdf" -F "dir=docs" http://127.0.0.1:1900/api/v1/storage
# => {"key":"docs/202609/ab12cd....pdf","url":"/api/v1/storage/docs/202609/ab12cd....pdf","size":12345}

# Download / delete
curl -O http://127.0.0.1:1900/api/v1/storage/docs/202609/ab12cd....pdf
curl -X DELETE http://127.0.0.1:1900/api/v1/storage/docs/202609/ab12cd....pdf
```

See the full [storage guide](docs/storage.md).

## Deployment

The binary is self-contained. Build a pure-Go image (no CGO toolchain) with:

```bash
just docker        # or: docker build -t airway .
docker run -p 1900:1900 -e AIRWAY_ENV=production -e DSN="sqlite:///app/tmp/airway.db" airway
```

Run migrations before/after deploy:

```bash
./airway cli db:migrate
```

See [docs/docker-compose.yml.example](docs/docker-compose.yml.example) for a
compose example. To serve the app under a path prefix behind a reverse proxy,
set `URL_PREFIX` (e.g. `/airway`) — see [HTTP endpoints](#http-endpoints).

## Guides

- [CLI scaffolding guide](docs/cli.md) · [文件存储指南](docs/storage.md)
- [templ views guide](docs/template.md)
- [SQL Builder DSL 指南（中文）](docs/zh-CN/sql-builder.md)
- [中文文档](docs/zh-CN/README.md)
