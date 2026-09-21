# Plan: Config-driven Admin Backend Generator (`admin:generate`)

Generate a complete admin panel — authentication, a dashboard with sidebar
navigation, and full CRUD for every table — from a single TOML file.

## Confirmed decisions

| Decision | Choice |
|---|---|
| Approach | **Static code generation** (Rails-style: generated code is real, user-owned code) |
| Auth | **Built-in login**: `admin_users` + cookie sessions + middleware |
| Field types | Base 5 (string/text/integer/float/boolean) **plus** datetime, enum, references, attachment |

## TOML format

Default path: `config/admin.toml` (overridable: `airway admin:generate [path]`).

```toml
[category]
name = "string"
sort_order = "integer"

[post]
title = "string"
body = "text"
published = "boolean"
published_at = "datetime"
status = "enum:draft,published,archived"
category_id = "references"                # infers target [category] from the _id suffix
cover = "attachment"
author_id = "references:member"           # explicit target (a [member] table)
```

Rules:

- Each top-level table is one resource. **Keys are singular resource names**
  (same convention as `generate scaffold post`); the SQL table name is the
  pluralized form via the existing `pluralize()`.
- Field names are `snake_case` (they become column names verbatim).
- Reserved/generated columns per table: `id`, `created_at`, `updated_at` —
  error if the TOML declares them.
- Type grammar:

  | TOML type | Go model field | SQL column | Form control (island) |
  |---|---|---|---|
  | `string` | `string` | `VARCHAR(255)` | Input |
  | `text` | `string` | `TEXT` | Textarea |
  | `integer` / `int` | `int64` | `BIGINT` | Input (number) |
  | `float` | `float64` | `DOUBLE PRECISION` | Input (number) |
  | `boolean` / `bool` | `bool` | `BOOLEAN` | Checkbox |
  | `datetime` | `*time.Time` | `TIMESTAMP` | Input (datetime-local) |
  | `enum:a,b,c` | `string` | `VARCHAR(255)` + `CHECK` | Select |
  | `references` / `references:name` | `int64` | `BIGINT` + `FOREIGN KEY` + index | Select (options loaded from the target's list API) |
  | `attachment` | `string` (stored URL) | `VARCHAR(255)` | File input → uploads via admin storage endpoint |

- Unknown types, empty tables, duplicate table/field names, `references`
  pointing at an unknown table, and circular references are hard errors.

## Generated artifacts

For `[post]` above (first run, alongside the other tables):

```
app/
  models/
    admin_user.go              # auth: email + password_digest (once)
    admin_session.go           # auth: token + user_id + expires_at (once)
    category.go                # per TOML table
    post.go                    #   incl. Relations() for references fields
  middlewares/
    admin_auth.go              # cookie → session → c.Set("admin_user") (once)
  api/admin_api/
    routes.go                  # static: mounts registry + auth + login routes (once)
    registry.go                # AdminResource type + registerResource() (once)
    auth_action.go             # login/logout page actions (once)
    uploads_action.go          # POST /api/v1/admin/uploads (attachment upload) (once)
    openapi.go                 # auth + uploads declarations (once)
    post_resource.go           # init() self-registration + CRUD actions + OpenAPI decls
  views/admin/                 # single package for the whole admin module
    admin.templ                # AdminLayout(title, active, nav) — sidebar shell (once)
    index.templ                # dashboard: one card per registered resource (once)
    login.templ                # server-rendered login form, no island needed (once)
    post_index.templ           # per-table page hosting its CRUD island
  assets/js/islands/
    admin-post-crud.tsx        # per-table island (DataTable + modal form, type-aware)
db/migrate/
  <ts>_create_admin_tables.up.sql/.down.sql   # one pair per generator run
config/routes.go               # edited once: admin_api.Routes(r) before plugin.MountAll
```

### Registry pattern (additive re-runs)

Re-running `admin:generate` after adding a table to the TOML must be purely
additive — no edits to already-generated files. Each per-table file
self-registers:

```go
// post_resource.go
func init() {
    registerAdminResource(adminResource{
        SlugPlural: "posts", Name: "Post", NamePlural: "Posts",
        Label: "Posts", Mount: mountPostRoutes, Page: PostPageAction,
    })
}
```

`routes.go` (generated once) iterates the registry to mount everything under
`/admin` and `/api/v1/admin`; the sidebar and dashboard also render from the
registry, so newly added tables appear automatically after a re-run.

### URL map

| Route | Purpose |
|---|---|
| `GET /admin` | dashboard |
| `GET /admin/login` · `POST /admin/login` · `POST /admin/logout` | auth (server-rendered form posts) |
| `GET /admin/<plural>` | per-table page |
| `GET/POST /api/v1/admin/<plural>`, `PUT/DELETE /api/v1/admin/<plural>/:id` | CRUD JSON API |
| `POST /api/v1/admin/uploads` | multipart upload via `storage.Current()`, returns URL |

All of the above except the login routes sit behind the `AdminAuth`
middleware.

## Authentication design

- `admin_users`: `email VARCHAR(255) UNIQUE`, `password_digest VARCHAR(255)`
  (hashed with the existing `utils.EncryptPassword`).
- `admin_sessions`: `token VARCHAR(64)` PK (`utils.RandomHex(32)`),
  `admin_user_id` FK, `expires_at TIMESTAMP`, `created_at`.
- Cookie `airway_admin_session`: HttpOnly, SameSite=Lax, Path=/, Secure
  outside `AIRWAY_ENV=local`, 7-day expiry. Logout deletes the row;
  the middleware opportunistically deletes expired sessions.
- **No first-visit bootstrap page** (attacker-on-empty-DB risk). Accounts are
  created by a new framework CLI command:

  ```bash
  airway admin:user admin@example.com 's3cret'   # proxied to `go run .` in projects
  ```

  It opens the DSN like `db:migrate` does and inserts into `admin_users` via
  the map-based repo API (`repo.InsertMap`), so the framework CLI needs no
  import of project code.
- Middleware behavior: API routes get 401 JSON; pages redirect to
  `/admin/login`.

## Generator mechanics

- Parse TOML with `pelletier/go-toml/v2` (promote from indirect to direct
  dependency — already in the module graph).
- **Migration ordering**: one migration pair per generator run
  (`<ts>_create_admin_tables`), containing auth tables (first run only) and
  the TOML tables in topological order of `references` dependencies. FKs use
  portable table-constraint syntax
  (`FOREIGN KEY (category_id) REFERENCES categories(id)`) plus an index on
  each FK column; enums get a `CHECK (col IN (...))` constraint.
- **Idempotency**: every target file is written only if absent
  (`writeTemplateFile` already refuses to overwrite); existing files are
  reported as `skipped`. A table whose `app/models/<name>.go` already exists
  is skipped entirely with a warning (the generator does not parse existing
  models to retrofit admin pages — v1 limitation). `config/routes.go` is
  edited only on the first run (same anchor-based edit as
  `registerScaffoldRoutes`).
- Tables removed from the TOML are **not** deleted; after generation the code
  is the source of truth. The run summary makes this explicit.
- Update semantics: changing a field type in the TOML does not alter existing
  files or generate ALTERs (scaffold parity — hand-edit or write a migration).
- Per-table code largely extends the existing scaffold templates
  (model/actions/openapi/island); the admin generator reuses `pluralize`,
  `toCamelName`, `idColumnForDSN`, `currentModulePath`, and the route-wiring
  helper.

## Frontend behavior

- Admin pages use a dedicated `AdminLayout` (sidebar + content), independent
  of `layouts.Base`.
- Each island is the scaffold CRUD island extended with type-aware controls:
  Select for enum, datetime-local input for datetime, remote Select for
  references (label = first `name`/`title`/`label`/`email` field of the
  target, else `#id`), file picker + upload for attachment (shows the stored
  URL), Checkbox for boolean. `datetime` fields serialize as RFC 3339 or
  `null`; the table view formats them.
- After generation: `go generate ./...` (templ), `go run . js:build`, commit
  the refreshed `*_templ.go` and `dist/` per repo convention.

## Implementation phases

1. **Type system + TOML parsing** (`cmd/cli_admin.go`): config structs, type
   grammar parsing, validation (reserved names, unknown refs, cycles),
   topological sort, migration SQL rendering. Unit tests for all of it.
2. **Generator core**: per-table templates (model/resource actions/OpenAPI/
   views/islands), registry + once-only files, `admin:generate` dispatch,
   `config/routes.go` wiring, run summary output. Tests over a tmp dir.
3. **Auth**: models, migrations, `AdminAuth` middleware, login/logout
   actions + login view, `airway admin:user` command. Tests against SQLite
   in-memory (follow `lib/repo` integration-test style where a DB is needed).
4. **Frontend**: AdminLayout/dashboard/login views, type-aware island
   controls, admin uploads endpoint.
5. **Docs & verification**: update `docs/cli.md` (+ zh-CN counterpart),
   `AGENTS.md` CLI section; full e2e pass — scaffold a temp project, run the
   generator covering every field type, `go build`, `db:migrate`,
   `admin:user`, login, exercise CRUD incl. an attachment upload (local
   storage); `gofmt`, `go vet ./...`, `go test ./...` with zero no-test
   packages.

## Follow-ups (implemented)

All of the v1 follow-ups have shipped:

- Server-side search/filter/pagination/sort — list API takes `page`,
  `page_size`, `q`, `sort`/`order` (whitelisted) and exact-match field
  filters; the island ships a search box, pagination and Export CSV.
- CSRF tokens on the login form (double-submit cookie) and login rate
  limiting (`lib/ratelimit`, 5 failures / 15 min per IP+email).
- Roles — `admin_users.role` (admin/editor/viewer) with `AdminRequireWrite`
  and `AdminRequireAdmin` middleware; `admin:user <email> <password> [role]`.
- Audit log — `admin_audit_logs` written on every mutation, reviewed at
  `/admin/audit-log` (admin only). Old installs get an upgrade migration.
- Soft delete — declaring `deleted_at = "datetime"` opts a table in.
- Display labels / i18n — optional `[table.meta]` (`label`, `labels`).
- Re-customizing generated tables — `admin:generate --force[=tables]`.
