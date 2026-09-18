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
# 1. Enable the plugin, copy its embedded SQL migrations into db/migrate,
#    and merge its deps/ directory into the project's deps/ directory.
#    This single command runs `go get`, adds the blank import to plugins.go,
#    and installs the migrations and deps/ in the same process.
go run . plugin:install github.com/example/airway-im-plugin

# 2. Migrate as usual
go run . db:migrate
```

Handy commands:

```bash
go run . plugin:list              # registered plugins and their mount paths
go run . plugin:install <module>  # enable a plugin, install its SQL migrations and deps/
```

`plugin:install` enables the plugin for you: if the plugin is not compiled
into the current binary, it adds the blank import to plugins.go, runs
`go get <module>` (or wires a local directory through a `replace` directive),
then reads the plugin's migrations and deps/ from its module directory on
disk — everything happens in the same process, so the current CLI's installer
logic is always the one used. You can still do these steps by hand if you
prefer.

`plugin:list` only sees plugins compiled into the running binary. Inside a
project the globally installed `airway` proxies to `go run .` automatically
(it prints a `proxying to project binary` notice), so the project's own
plugins are what gets listed.

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
`app/models/` and `host/db/migrate/` directories, then runs `go mod tidy`.

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
  host/                   # optional: files mirrored into the host project's
                          # own tree by `plugin:install`, preserving relative
                          # paths (host/Caddyfile → the host's Caddyfile)
    db/migrate/           # SQL migrations (*.up.sql / *.down.sql) — copied
                          # with fresh timestamps, not mirrored verbatim
  deps/                   # optional: extra files merged into the host
                          # project's deps/ directory by `plugin:install`;
                          # namespace them under the plugin name
                          # (deps/im/app/... for a plugin named im)
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
//go:embed host/db/migrate
var migrations embed.FS

func (IMPlugin) MigrationFS() fs.FS { return migrations }
```

REPL model names must not collide with host models or other plugins' models;
conflicts disable plugin REPL models and log a warning.

### 3. Migrations — two styles

- **Migrations written in Go code** need no install step: call
  `schema.RegisterChange` from `lib/migrate/schema` in an `init()` (exactly
  like a host app's Go migrations) and they join the global migration list
  on import.
- **SQL files** (`<version>_<name>.up.sql` / `.down.sql`) live under the
  plugin's `host/db/migrate/` directory. They are either embedded via
  `MigrationFS()` or read from the module directory on disk, and copied into
  the host's `db/migrate/` by `plugin:install <module>` (run as
  `go run . plugin:install <module>` in the host project) with fresh
  timestamps. After copying they
  are ordinary host migrations: `db:migrate`, `db:rollback` and `db:status`
  work on them unchanged, and re-running `plugin:install` skips files already
  installed.

### 4. Shipping extra project files with `deps/` and `host/`

Everything under the plugin's top-level `deps/` directory is merged into the
host project's `deps/` directory by `plugin:install` — use it for companion
services, deploy configs, or any files the host project needs on disk.
Namespace the files under your plugin's name (`deps/im/app/...` for a plugin
named `im`) so several plugins can install side by side without collisions.
Existing destination files are skipped (never overwritten), so re-running
`plugin:install` is safe; delete the installed copy first if you want to
refresh it from a newer plugin version.

Files that must land at the host project root instead — a deploy
`docker-compose.yml`, a `Containerfile`, a config the host builds on — go
under the plugin's `host/` directory, which `plugin:install` mirrors into the
host's own tree preserving relative paths (`host/docker-compose.yml` becomes
the host's `docker-compose.yml`). The same rules apply: existing files are
skipped, and a single `.templ` suffix is stripped. The `host/db/migrate`
subtree is exempt from this mirroring — it belongs to the migration
installer above.

For a Compose stack, cascade rather than collide: keep the plugin's own
services in `deps/<name>/docker-compose.yml` (build contexts in it resolve
relative to that file when included) and ship a root-level
`docker-compose.yml` via `host/` that pulls it in with Compose's `include`
directive — a host that already has its own compose file keeps it (the
install skips) and adds the same `include` line by hand.

The scaffolded host project ships an empty `deps/` directory; the install
creates it when missing, so projects scaffolded by older Airway versions work
unchanged.

Two Go module rules shape what you can ship:

- **No nested `go.mod` inside `deps/`.** Module zips drop nested modules
  entirely, so a real `go.mod` would never reach the host. Ship it as
  `go.mod.templ` instead — the install strips one `.templ` suffix, restoring
  `go.mod` in the host project. Keeping the real `go.mod` beside its `.templ`
  variant so the nested module still builds in your checkout is fine: the
  install ships the `.templ` content only (the bare file is skipped), matching
  what a module-zip download would deliver — and a local-directory install
  fails fast when the bare `go.mod` has drifted from its `.templ`. The suffix
  matches the templ engine's name, leaving room for the install to render
  such files as templates in the future. `go.sum` triggers no such rule: ship
  it under its own name.
- **Never name the directory `vendor/`.** Module zips drop `vendor/`
  wholesale, which is why the convention lives in `deps/`.

One caveat if your plugin also uses templ views: `templ generate` parses
every `.templ` file under its working directory, including `deps/`. Scope the
generate directive to the views directory
(`//go:generate go tool templ generate -path app/views`) so install templates
are left alone.

Don't commit build artifacts (compiled binaries, caches) under `deps/` — they
would be merged into every host project's `deps/`.

### 5. Views and WebSocket

- templ views compile to Go, so a plugin keeps its own `app/views/` package
  and commits the generated `*_templ.go` files — no special handling needed.
- Plugins may import `github.com/daqing/airway/app/websocket` to publish
  real-time events through the host's hub.

### 6. Framework packages available to plugins

Everything under `lib/` (`repo`, `sql`, `render`, `storage`, `validation`,
`utils`, ...) plus `app/websocket` can be imported from a plugin module via
`github.com/daqing/airway/...`.
