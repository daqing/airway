# Static Showcase Sites (SSG) — Design Record

Airway doubles as a static site generator for showcase websites — company
homepages, product landings, portfolios. A site is a regular Go project: it
declares its pages in a root-package `ssg.go` against a swappable **theme
module**, and `airway ssg:build` exports them as a plain HTML directory
deployable to any static host. Pages render with the same
[templ](https://templ.guide) components the framework uses for
server-rendered views — no database, no Node toolchain, no running server.

Status: **implemented and verified end to end (2026-09-22)** — the full
lifecycle `theme:new` → `theme:install` → `ssg:new` → `ssg:build` →
`ssg:serve` runs green on fresh scaffolds.

## How it works

```
ssg.go (project root)                 theme module (independent Go module)
  cmd.SetSSGBuilder(init())             implements lib/ssg.Theme
  ssg.New(meta).Use(theme)              templ layout + page components
  s.Page(slug, title, data)             go:embed assets (CSS, fonts)
        │                                      ▲
        │  airway ssg:build                    │ consumed as a module dep
        ▼                                      │ (require, or replace+require)
  lib/ssg.Build: render every page ───► dist/<slug>/index.html
                 copy theme assets ───► dist/assets/
```

- The project's root-package `ssg.go` registers a **builder** with
  `cmd.SetSSGBuilder` in `init()` — the same compile-time registration
  pattern plugins and REPL models use. `ssg:build` and `ssg:serve` run on
  whatever that builder returns, so the project binary carries its own site
  code without any `main.go` changes.
- `lib/ssg.Build` renders each registered page through the theme into
  `<out>/<slug>/index.html` (`/` → `index.html`, `/about` →
  `about/index.html`) and copies the theme's embedded assets into
  `<out>/assets/`.
- `lib/ssg.Handler` serves the same site dynamically for `ssg:serve`: pages
  render per request, assets stream from the theme's embedded filesystem.

## Design decisions

1. **Static export over dynamic hosting.** Showcase sites want static
   hosting, SEO, and zero ops. templ components render to any `io.Writer`
   without an HTTP request — the same primitive `render.HTML` uses — so the
   export engine is a thin loop, and `ssg:serve` reuses the same rendering
   for preview.
2. **Themes as Go modules, not template directories.** Unlike Hugo/Jekyll
   theme folders, a theme here is compiled, type-checked, and unit-tested
   Go code; distribution and versioning reuse the module system (and
   `theme:install` mirrors `plugin:install` in spirit).
3. **Engine in the core, themes outside.** CLI dispatch is compiled into
   the binary, so plugins cannot add commands — `ssg:build`/`ssg:serve`/
   `ssg:new`/`theme:*` therefore live in `cmd/` with the engine in
   `lib/ssg`. Themes themselves need zero framework changes, which the
   standalone `airway-terminal-theme` demonstrates.
4. **Compile-time registration over config.** `cmd.SetSSGBuilder` from an
   `init()` keeps the Rails-y "code over config" stance: pages are Go
   function calls, and nothing gets parsed at build time.

## Implementation map

| Piece | Location |
|---|---|
| Site/Page/Theme model, slug normalization, builder type | `lib/ssg/ssg.go` |
| Static exporter + preview handler | `lib/ssg/build.go` |
| `ssg:build`, `ssg:serve`, `cmd.SetSSGBuilder` | `cmd/cli_ssg.go` |
| `ssg:new` scaffold | `cmd/cli_ssg_new.go` + `cmd/clitemplate/ssgtemplate/` |
| `theme:new` scaffold | `cmd/cli_theme_new.go` + `cmd/clitemplate/themetemplate/` |
| `theme:install` | `cmd/cli_theme.go` |
| Bundled reference theme | `themes/corporate/` (separate module, `replace` to the checkout) |
| Usage guide | [docs/ssg.md](docs/ssg.md) / [docs/zh-CN/ssg.md](docs/zh-CN/ssg.md) |

## Verification record

- **Unit tests** (table-driven, no network): `lib/ssg` (export layout,
  preview handler, slug normalization, registration panics), `cmd` (site
  scaffold wiring for both local-replace and pinned go.mod forms, theme
  package derivation, theme install replace/require plus its validation
  errors, proxy exemptions).
- **End to end** (2026-09-22): `ssg:new` → `ssg:build` (slug→path mapping,
  asset copy, generated markup) → `ssg:serve` verified with curl (pages and
  assets 200, unknown paths 404); `theme:new` scaffold built and tested green
  immediately; `theme:install` of a local theme followed by an `s.Use` swap
  changed the built site's assets to the installed theme's.
- **Gates**: `go vet ./...`, `go test ./...` (zero packages without tests),
  `gofmt` clean; theme modules are separate Go modules and run their tests
  from their own directories.

> 中文版:[SSG.zh-CN.md](SSG.zh-CN.md)
> Usage guide (quick start, commands, page definition, themes, deployment):
> [docs/ssg.md](docs/ssg.md)
