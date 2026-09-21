# Desktop Apps with Wails

Airway projects can be exported into a [Wails v3](https://v3.wails.io) desktop
application: the same web stack (templ views, islands, JSON APIs, WebSocket)
runs on a random `127.0.0.1` port inside a desktop process, and a native
WebView window loads that address. Server-rendered pages, cookie sessions,
redirects and WebSockets all behave exactly as on the web — the desktop shell
is additive, and no application code changes.

The research and decision record behind this integration lives in
[WAILS.md](../WAILS.md) at the repository root.

## How it works

```
┌─ desktop binary ─────────────────────────────────────────┐
│ boot.New: SQLite in the user data dir → embedded         │
│           migrations → local storage → plugins → routes  │
│ http.Server on 127.0.0.1:<random port>                   │
│ Wails v3 window → loads http://127.0.0.1:<port>          │
└──────────────────────────────────────────────────────────┘
```

- **Database:** pure-Go SQLite at
  `<UserConfigDir>/<AppName>/data.db` (e.g.
  `~/Library/Application Support/<AppName>/` on macOS,
  `%APPDATA%\<AppName>` on Windows). Migrations from `db/migrate`
  are embedded into the binary and applied automatically on startup.
- **File storage:** the local driver, rooted at
  `<UserConfigDir>/<AppName>/storage`.
- **Hardening:** the desktop boot pins `AIRWAY_ENV=production`, serves the
  committed frontend bundle (no Node, no dev server), replaces the permissive
  CORS middleware with same-origin-only checks, and restricts WebSocket
  upgrades to same-origin requests.
- **The window is a real browser view** pointed at your own server, so
  same-origin fetches (`apiFetch`), cookies, 302 redirects and
  `ws://127.0.0.1:<port>/ws` all work unchanged.

## Generate the desktop target

Inside a project:

```bash
airway desktop:init
```

This generates:

```
desktop/
  main.go            # Wails window loading the local server (user-owned after first run)
  migrations.go      # go:embed of migrations + plugin mirror (regenerated on re-run)
  migrations/        # synced copy of db/migrate/*.sql
  Taskfile.yml       # build / package / run / dev entry points
  build/             # icons, plists, installer configs (Wails build system)
```

`desktop:init` also runs `go get github.com/wailsapp/wails/v3@v3.0.0-beta.24`
to pin the Wails module in the project's go.mod. Re-running the command is
safe: it refreshes `desktop/migrations/` and the plugin mirror but never
touches `desktop/main.go` unless you pass `--force`.

Plugins declared in the project's root `plugins.go` are mirrored as blank
imports into `desktop/migrations.go`, so the desktop binary compiles the same
plugins as the web binary. Add a plugin with `plugin:install`, then re-run
`airway desktop:init`.

## Build and package

Install the toolchain once:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
go install github.com/go-task/task/v3/cmd/task@latest
```

Then, from the `desktop/` directory:

```bash
wails3 task build        # native build for the current OS
wails3 task package      # macOS .app / Windows NSIS installer / Linux deb+rpm
wails3 task dev          # run with rebuild-on-change
```

Outputs land in `desktop/bin/`. The macOS build targets macOS 12+; the macOS
`.app` is ad-hoc signed — Developer ID signing and notarization are required
for distribution outside your machine (`wails3 task darwin:sign` /
`darwin:sign:notarize`).

### Platform notes

| Platform | Requirements | Notes |
|---|---|---|
| macOS 12+ | Xcode command line tools | universal binary: `wails3 task darwin:package:universal` |
| Windows 10/11 | WebView2 runtime (auto-installed by the NSIS installer) | cross-compiles from macOS/Linux without CGO |
| Linux | GTK4 + WebKitGTK 6.0 (Ubuntu 24.04+/Debian 13+); `-tags gtk3` for the legacy WebKit2GTK 4.1 stack | deb/rpm/AppImage packaging via nfpm |

Cross-compilation for macOS and Linux CGO builds uses the Wails Docker image
(`wails3 task setup:docker`, ~800 MB download). For production releases a
GitHub Actions matrix (one job per OS) is the recommended route.

## Data, migration and upgrade semantics

- `desktop/migrations/` is a **copy** of `db/migrate/`. Re-running
  `airway desktop:init` re-syncs it; migrations removed from the source
  directory are deliberately kept in the copy (the command never deletes), so
  old desktop installs still find them.
- Migrations apply automatically at every startup; `schema_migrations`
  tracks applied versions, so first launch on a fresh machine creates the
  full schema.
- `db/schema.json` is not written by desktop runs (snapshot IO is disabled in
  `boot.Options`).

## Limitations

- Experimental until Wails v3 reaches GA; the Wails version is pinned and
  should be upgraded deliberately.
- The localhost port is reachable by other processes of the same user — the
  same trust model as every local-server desktop app. Same-origin hardening
  is enabled, but do not assume the port is secret.
- External `https://` links navigate the desktop window itself; route them to
  the system browser from your UI code if that is not desired.
