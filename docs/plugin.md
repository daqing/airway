# Airway Plugins

Plugins are Airway's extension mechanism, named after the WordPress plugin
model. A Plugin is a self-contained feature module — routes, models,
migrations, views — shipped as an **independent Go module** (typically its own
git repository). A host application installs one with `go get` and enables it
with a single blank import. Not every project needs every feature: keep your
app lean and pull in a Plugin (IM, admin panel, billing, ...) only when you
need it.

## Using a Plugin (host application)

```bash
# 1. Install the module
go get github.com/example/airway-im-plugin

# 2. Enable it — add a blank import to plugins.go (project root, package main):
#    import (
#        _ "github.com/example/airway-im-plugin"
#    )

# 3. Copy the plugin's embedded SQL migrations into db/migrate (if it ships any)
go run . plugin:install im

# 4. Migrate as usual
go run . db:migrate
```

Handy commands:

```bash
go run . plugin:list            # registered plugins and their mount paths
go run . plugin:install [name]  # copy a plugin's SQL migrations
```

Plugins register at compile time, so run these commands through the project
binary (`go run . ...` in the project directory): a globally installed `airway`
CLI can only list and install the plugins compiled into itself.

Routes registered by the plugin answer under its declared mount path (e.g.
`/api/v1/im`). Plugin models that opt in appear in `go run . repl` alongside
your own models.

To take over a plugin's mount path yourself, skip `plugin.MountAll` for it and
mount manually in `config/routes.go`:

```go
myplugin.Plugin.Routes(r.Group("/custom/prefix"))
```

## Authoring a Plugin

Scaffold a new plugin module with the CLI (works with the globally installed
`airway` — no compile-time registration involved):

```bash
airway plugin:new im                              # directory: im
airway plugin:new github.com/me/airway-im-plugin  # plugin name derived from
                                                  # the last path segment
```

This generates `go.mod`, `plugin.go` (Plugin implementation + `init()`
registration), a sample API module under `app/api/<name>_api/`, and empty
`app/models/` and `db/migrate/` directories, then runs `go mod tidy`.

A Plugin repository mirrors the layout of a regular Airway project:

```
airway-im-plugin/
  go.mod                  # module github.com/example/airway-im-plugin
                          # requires github.com/daqing/airway
  plugin.go               # Plugin implementation + init() registration
  app/
    api/im_api/           # routes + actions, same conventions as a host app
    models/               # model structs with db tags and TableName()
    views/                # templ views (commit the generated *_templ.go)
  db/
    migrate/              # optional: embedded *.up.sql / *.down.sql files
```

### 1. Implement and register the Plugin

```go
package implugin

import (
    "github.com/daqing/airway/lib/plugin"
    "github.com/example/airway-im-plugin/app/api/im_api"
    "github.com/gin-gonic/gin"
)

type IMPlugin struct{}

func (IMPlugin) Name() string      { return "im" }
func (IMPlugin) MountPath() string { return "/api/v1/im" }
func (IMPlugin) Routes(r *gin.RouterGroup) {
    im_api.Routes(r)
}

func init() {
    plugin.Register(IMPlugin{})
}
```

The plugin's package `init()` calls `plugin.Register`, so a blank import in the
host's `plugins.go` is all it takes to enable it. `Register` panics on a
duplicate name or a mount path that does not start with `/`.

### 2. Optional capabilities

Implement any of these interfaces and the framework picks them up
automatically:

```go
// Bootable — runs after DB/Redis/storage are ready, before the server starts.
// Use repo.CurrentDB(), storage.Current(), redis_client.Current() here.
func (IMPlugin) Boot() error { ... }

// REPLModelProvider — exposes models to `airway repl`.
func (IMPlugin) REPLModels() map[string]any {
    return map[string]any{"Message": models.Message{}}
}

// MigrationProvider — ships SQL migrations embedded in the binary.
//
//go:embed db/migrate
var migrations embed.FS

func (IMPlugin) MigrationFS() fs.FS { return migrations }
```

REPL model names must not collide with host models or other plugins' models;
conflicts disable plugin REPL models and log a warning.

### 3. Migrations — two styles

- **Go DSL migrations** need no install step: call `schema.RegisterChange` from
  `lib/migrate/schema` in an `init()` (exactly like a host app's DSL
  migrations) and they join the global migration list on import.
- **SQL files** (`<version>_<name>.up.sql` / `.down.sql`) are embedded via
  `MigrationFS()` and copied into the host's `db/migrate/` by
  `plugin:install <name>` (run as `go run . plugin:install <name>` in the host
  project) with fresh timestamps. After copying they
  are ordinary host migrations: `db:migrate`, `db:rollback` and `db:status`
  work on them unchanged, and re-running `plugin:install` skips files already
  installed.

### 4. Views and WebSocket

- templ views compile to Go, so a plugin keeps its own `app/views/` package
  and commits the generated `*_templ.go` files — no special handling needed.
- Plugins may import `github.com/daqing/airway/app/websocket` to publish
  real-time events through the host's hub.

### 5. Framework packages available to plugins

Everything under `lib/` (`repo`, `sql`, `render`, `storage`, `validation`,
`utils`, ...) plus `app/websocket` can be imported from a plugin module via
`github.com/daqing/airway/...`.
