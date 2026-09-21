# The Admin Panel

Airway can generate a complete, production-ready admin panel from a single
TOML file. One command produces authentication with roles, an audit trail, a
dashboard with sidebar navigation, and type-aware CRUD — server-rendered
pages plus Preact islands talking to a JSON API — for every table you
declare.

> 中文版文档:[ADMIN.zh-CN.md](ADMIN.zh-CN.md)

## Quick start

1. Describe your tables in `config/admin.toml` (see [The config file](#the-config-file)).
2. Run the generator and the standard follow-ups:

```bash
airway admin:generate                     # or: admin:generate --force=table1,table2
airway templates:compile                  # compile the .templ views (shorthand for `go generate ./...`)
airway js:build                           # bundle the CRUD islands
airway db:migrate                         # create the tables
airway admin:root admin 's3cret'             # create the administrator account
airway server                             # visit /admin
```

3. Sign in at `/admin/login` and you land on the dashboard: one card per
   resource, sidebar navigation, and full CRUD for every table.

## The config file

The generator reads `config/admin.toml` by default; any other path can be
passed as the only argument. Each top-level table is one resource, keyed by
its **singular** name — the SQL table name is the pluralized form
(`[post]` → `posts`). Every field maps a column name to a type:

```toml
[category]
name = "string"
sort_order = "integer"
parent_id = "references:category"    # self-reference (explicit target)

[member.meta]                        # optional display labels (any language)
label = "成员管理"
labels = { name = "姓名", email = "邮箱" }

[member]
name = "string"
email = "string"

[post]
title = "string"
body = "text"
views = "integer"
score = "float"
published = "boolean"
published_at = "datetime"
status = "enum:draft,published,archived"
cover = "attachment"
category_id = "references"           # target inferred from the _id suffix
deleted_at = "datetime"              # opts this table into soft deletes
```

### Field types

| TOML type | Go model field | SQL column | Form control |
|---|---|---|---|
| `string` | `string` | `VARCHAR(255)` | text input |
| `text` | `string` | `TEXT` | textarea |
| `integer` / `int` | `int64` | `BIGINT` | number input |
| `float` | `float64` | `DOUBLE PRECISION` | number input |
| `boolean` / `bool` | `bool` | `BOOLEAN` | checkbox |
| `datetime` | `*time.Time` | `TIMESTAMP` | datetime-local input |
| `enum:a,b,c` | `*string` | `VARCHAR(255)` + `CHECK` | select |
| `references[:table]` | `*int64` | `BIGINT` + FK + index | remote select |
| `attachment` | `string` | `VARCHAR(255)` | file upload |

Notes:

- `id`, `created_at` and `updated_at` are added to every table
  automatically and must not be declared.
- Table and field names must be lowercase snake_case identifiers. The
  singular key pluralizes to the SQL table; two tables mapping to the same
  plural are a hard error.
- `enum` values round-trip as `*string`, so clearing the select stores SQL
  NULL; the generated `CHECK` constraint rejects anything outside the list.
- `references` infers the target from the `_id` suffix
  (`category_id` → `category`) or takes an explicit one
  (`references:member`). Forward references work; circular references
  between tables are rejected. Fields also get a `Relations()` entry
  (`repo.NewBelongsTo`) for `repo.Preload`.
- Declaring `deleted_at = "datetime"` opts the table into **soft deletes**
  (see [Soft delete](#soft-delete)).

### `[table.meta]` — display labels

An optional `[table.meta]` sub-table overrides the generated display text
without touching the `field = type` format:

- `label` — the sidebar entry, page heading and dashboard card.
- `labels` — per-field overrides used in table columns, form labels and the
  CSV header.

The panel therefore speaks whatever language your TOML speaks.

## What the generator produces

For each table in the TOML:

- **Model** — `app/models/<name>.go` with `db`/`json` tags and REPL
  registration; nullable kinds map to pointers (`*string`, `*int64`,
  `*time.Time`).
- **Resource** — `app/api/admin_api/<name>_resource.go`: a self-registering
  file with the CRUD actions, the list-condition builder, the CSV renderer
  and OpenAPI declarations.
- **Page** — `app/views/admin/<plural>_page.templ`, hosting the CRUD island.
- **Island** — `app/assets/js/islands/admin-<plural>-crud.tsx`: DataTable +
  modal form with type-aware controls (enum select, remote reference select,
  datetime picker, checkbox, file upload).

Once per project (written only when absent):

- `app/models/admin_user.go`, `admin_session.go` — authentication.
- `app/middlewares/admin_auth.go` — `AdminAuth`, `AdminRequireWrite`,
  `AdminRequireAdmin`.
- `app/api/admin_api/` — `routes.go`, `registry.go` (shared helpers:
  paging, ordering whitelists, filters, CSV cells, audit writer),
  `auth_action.go` (sign-in/out, CSRF, rate limiting), `audit_action.go`,
  `uploads_action.go` (attachments through `storage.Current()`),
  `openapi.go`.
- `app/views/admin/` — `admin.templ` (sidebar layout), `index.templ`
  (dashboard), `login.templ`, `audit_log.templ`, `types.go`.
- One migration pair per run in `db/migrate/`
  (`<timestamp>_create_admin_tables.{up,down}.sql`): authentication tables
  on the first run, then every newly generated table in foreign-key
  dependency order, with FK constraints, FK indexes and enum `CHECK`
  constraints.

`config/routes.go` is wired once automatically (`admin_api.Routes(r)`),
with a printed snippet as fallback.

## Routes

| Route | Access | Purpose |
|---|---|---|
| `GET /admin` | any signed-in admin account | dashboard |
| `GET /admin/login` · `POST /admin/login` · `POST /admin/logout` | public | sign-in/out |
| `GET /admin/<plural>` | signed-in | resource page (CRUD island) |
| `GET /admin/audit-log` | `admin` role only | audit trail |
| `GET /api/v1/admin/<plural>` | signed-in | list (JSON) or CSV via `format=csv` |
| `POST /api/v1/admin/<plural>` | editor and above | create |
| `PUT /api/v1/admin/<plural>/:id` | editor and above | update |
| `DELETE /api/v1/admin/<plural>/:id` | editor and above | delete (or soft delete) |
| `POST /api/v1/admin/uploads` | editor and above | attachment upload |

When `URL_PREFIX` is set, every path above is served under the prefix and
redirects/links honor it automatically.

## Authentication and accounts

Accounts live in `admin_users` with bcrypt-hashed passwords. There are two
account commands:

```bash
airway admin:root <username> <password>                    # administrator (role admin)
airway admin:member <username> <password> [--role=editor|viewer]   # non-admin account
```

`editor` is the default role for `admin:member`; creating administrators
goes through `admin:root` only. Sessions are stored server-side in
`admin_sessions`: signing in issues a random 64-hex token delivered as the
`airway_admin_session` cookie (HttpOnly, `SameSite=Lax`, 7-day expiry,
`Secure` outside `AIRWAY_ENV=local`). There is no first-visit bootstrap
page — accounts exist only if you create them, so an empty deployment has
nothing to hijack.

## Roles

| Role | Read | Write (create/update/delete/upload) | Audit log |
|---|---|---|---|
| `admin` | ✓ | ✓ | ✓ |
| `editor` (default) | ✓ | ✓ | — |
| `viewer` | ✓ | — (403) | — (redirected) |

Mutating routes sit behind the `AdminRequireWrite` middleware, the audit
page behind `AdminRequireAdmin`. Unknown or missing roles are treated as
the least-privileged one.

## Security hardening

- **CSRF** — the sign-in form carries a random double-submit token; the
  POST is accepted only when the hidden field matches the
  `airway_admin_csrf` cookie (HttpOnly, `SameSite=Lax`, 12-hour expiry).
- **Rate limiting** — five failed sign-ins for the same IP and username pair
  lock that pair out for fifteen minutes (in-process fixed-window limiter,
  `lib/ratelimit`).
- **Audit log** — every create, update and delete records the acting
  account, the resource and the record id in `admin_audit_logs`. A failed
  audit write is logged but never blocks the business action. Admins review
  the trail at `/admin/audit-log` (newest first, paginated).
- **SQL safety** — list filters and ordering only accept identifiers from
  the generated whitelist; request values are always bound as parameters.

## The list API

`GET /api/v1/admin/<plural>` supports server-side everything:

| Parameter | Meaning |
|---|---|
| `page` | 1-based page number (default 1) |
| `page_size` | rows per page, 1–100 (default 20) |
| `q` | text search across string-ish fields (`LIKE`, `ILIKE` on PostgreSQL) |
| `sort` | column to order by — whitelisted per resource |
| `order` | `asc` (default) or `desc` |
| `<field>=<value>` | exact-match filter on any declared field, type-converted |
| `format=csv` | stream the matching rows as a CSV download instead of JSON |

Response (JSON):

```json
{ "code": 0, "data": { "items": [ ... ], "total": 42, "page": 1, "page_size": 20 } }
```

The CSV export uses the display labels as headers, quotes embedded
separators via `encoding/csv`, and neutralizes leading formula characters
(`=+-@`) so spreadsheets do not execute cell content. Filters, `q` and
ordering apply to the export as well.

The CRUD island ships the matching UI out of the box: a search box, a
pagination footer and an Export CSV link that preserves the active search.

## Soft delete

Adding `deleted_at = "datetime"` to a table changes its lifecycle:

- Delete becomes an UPDATE that stamps `deleted_at` (and skips already
  deleted rows).
- Every list/count read filters `deleted_at IS NULL`.
- The column stays on the model and in migrations, but is excluded from
  forms, params and CSV export.

Tables without the column keep hard deletes. Restoring a row is a
one-line SQL/REPL statement — there is no restore UI yet.

## Attachments

Attachment fields upload through `storage.Current()` — the same local/S3/R2/COS
configuration the rest of the app uses — and store the resulting URL in the
column. The island's file control uploads immediately on selection and shows
a link to the stored file.

## Re-running the generator

- Adding tables to the TOML is **additive**: existing files are never
  rewritten, and the self-registering resources make the sidebar, dashboard
  and routes pick up new tables automatically.
- **`--force`** (or `--force=table1,table2`) rewrites the generated files of
  existing tables — model, resource, page and island — discarding hand
  edits in those files. It never generates migrations, so schema changes
  still need a hand-written migration in `db/migrate/`.
- Removing a table from the TOML does not delete generated code, and
  changing a field's type does not alter existing files or migrations.
- Projects generated before roles/audit existed get an **upgrade
  migration** automatically on the next run (adds `admin_users.role` and
  creates `admin_audit_logs`).

## Customizing the output

Everything under `app/` is yours. The generated files carry a "edit freely"
note and are regenerated only when you ask for it (`--force`), so safe
customizations include:

- extending the resource actions (extra routes, validation, side effects) —
  the shared helpers in `registry.go` (`adminListPaging`, `adminListOrder`,
  `adminAudit`, …) are ordinary Go functions;
- restyling `app/views/admin/*.templ` and the `aw-admin-*` classes in
  `app/assets/css/airway.css`;
- tightening the OpenAPI declarations in each resource file.

Remember the project ritual after editing views or islands:

```bash
airway templates:compile && airway js:build
```

## Current limitations

- No in-panel user management: roles are assigned via `admin:root`.
- No soft-delete restore UI (SQL/REPL only).
- The rate limiter is per-process; multi-replica deployments should put a
  shared limiter in front.
- The audit log has no retention policy and no export.
