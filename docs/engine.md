# Airway Engines

Engines are Airway's extension mechanism, inspired by Rails Engines. An Engine
is a self-contained feature module — routes, models, migrations, views — shipped
as an **independent Go module** (typically its own git repository). A host
application installs one with `go get` and enables it with a single blank
import. Not every project needs every feature: keep your app lean and pull in an
Engine (IM, admin panel, billing, ...) only when you need it.

## Using an Engine (host application)

```bash
# 1. Install the module
go get github.com/example/airway-im-engine

# 2. Enable it — add a blank import to engines.go (project root, package main):
#    import (
#        _ "github.com/example/airway-im-engine"
#    )

# 3. Copy the engine's embedded SQL migrations into db/migrate (if it ships any)
go run . engine:install im

# 4. Migrate as usual
go run . db:migrate
```

Handy commands:

```bash
go run . engine:list            # registered engines and their mount paths
go run . engine:install [name]  # copy an engine's SQL migrations
```

Engines register at compile time, so run these commands through the project
binary (`go run . ...` in the project directory): a globally installed `airway`
CLI can only list and install the engines compiled into itself.

Routes registered by the engine answer under its declared mount path (e.g.
`/api/v1/im`). Engine models that opt in appear in `go run . repl` alongside
your own models.

To take over an engine's mount path yourself, skip `engine.MountAll` for it and
mount manually in `config/routes.go`:

```go
myengine.Engine.Routes(r.Group("/custom/prefix"))
```

## Authoring an Engine

An Engine repository mirrors the layout of a regular Airway project:

```
airway-im-engine/
  go.mod                  # module github.com/example/airway-im-engine
                          # requires github.com/daqing/airway
  engine.go               # Engine implementation + init() registration
  app/
    api/im_api/           # routes + actions, same conventions as a host app
    models/               # model structs with db tags and TableName()
    views/                # templ views (commit the generated *_templ.go)
  db/
    migrate/              # optional: embedded *.up.sql / *.down.sql files
```

### 1. Implement and register the Engine

```go
package imengine

import (
    "github.com/daqing/airway/lib/engine"
    "github.com/example/airway-im-engine/app/api/im_api"
    "github.com/gin-gonic/gin"
)

type IMEngine struct{}

func (IMEngine) Name() string      { return "im" }
func (IMEngine) MountPath() string { return "/api/v1/im" }
func (IMEngine) Routes(r *gin.RouterGroup) {
    im_api.Routes(r)
}

func init() {
    engine.Register(IMEngine{})
}
```

The engine's package `init()` calls `engine.Register`, so a blank import in the
host's `engines.go` is all it takes to enable it. `Register` panics on a
duplicate name or a mount path that does not start with `/`.

### 2. Optional capabilities

Implement any of these interfaces and the framework picks them up
automatically:

```go
// Bootable — runs after DB/Redis/storage are ready, before the server starts.
// Use repo.CurrentDB(), storage.Current(), redis_client.Current() here.
func (IMEngine) Boot() error { ... }

// REPLModelProvider — exposes models to `airway repl`.
func (IMEngine) REPLModels() map[string]any {
    return map[string]any{"Message": models.Message{}}
}

// MigrationProvider — ships SQL migrations embedded in the binary.
//
//go:embed db/migrate
var migrations embed.FS

func (IMEngine) MigrationFS() fs.FS { return migrations }
```

REPL model names must not collide with host models or other engines' models;
conflicts disable engine REPL models and log a warning.

### 3. Migrations — two styles

- **Go DSL migrations** need no install step: call `schema.RegisterChange` from
  `lib/migrate/schema` in an `init()` (exactly like a host app's DSL
  migrations) and they join the global migration list on import.
- **SQL files** (`<version>_<name>.up.sql` / `.down.sql`) are embedded via
  `MigrationFS()` and copied into the host's `db/migrate/` by
  `engine:install <name>` (run as `go run . engine:install <name>` in the host
  project) with fresh timestamps. After copying they
  are ordinary host migrations: `db:migrate`, `db:rollback` and `db:status`
  work on them unchanged, and re-running `engine:install` skips files already
  installed.

### 4. Views and WebSocket

- templ views compile to Go, so an engine keeps its own `app/views/` package
  and commits the generated `*_templ.go` files — no special handling needed.
- Engines may import `github.com/daqing/airway/app/websocket` to publish
  real-time events through the host's hub.

### 5. Framework packages available to engines

Everything under `lib/` (`repo`, `sql`, `render`, `storage`, `validation`,
`utils`, ...) plus `app/websocket` can be imported from an engine module via
`github.com/daqing/airway/...`.
