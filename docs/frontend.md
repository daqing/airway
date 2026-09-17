# Frontend Guide

Airway ships a complete frontend pipeline with **no Node.js toolchain**:
npm packages are fetched, bundled and served by the `airway` CLI and the Go
binary itself. Pages stay server-rendered with templ; interactive regions
are **islands** — Preact components mounted on small, well-defined spots of
the page.

- Chinese version: [docs/zh-CN/frontend.md](zh-CN/frontend.md)

## The big picture

| Layer | What it is |
| --- | --- |
| `js.pkg.json` | Dependency manifest: `deps` (exact versions, hand-editable) + `lock` (resolved tree with sha512 integrity). |
| `app/assets/js/vendor/` | Installed packages in node_modules layout (committed). |
| `app/assets/js/islands/` | Your interactive components; each file default-exports one island. |
| `app/assets/js/ui/` | airway-ui, the bundled component library. |
| `app/assets/css/airway.css` | Design tokens + component styles (light/dark). |
| `app/assets/dist/` | `js:build` output (committed) — embedded into the binary. |

## Commands

```bash
airway js:add preact                        # add a dependency (latest, pinned exactly)
airway js:add @tanstack/react-table@9.2.4   # add at an explicit version
airway js:install                           # install per js.pkg.json (idempotent)
airway js:build                             # bundle app/assets/js -> app/assets/dist
airway generate island chart                # scaffold an island component
airway generate scaffold post title:string  # full CRUD: model, migration, API, page, island
```

Registry: `AIRWAY_JS_REGISTRY` (default `https://registry.npmjs.org`; set a
mirror like `https://registry.npmmirror.com` when the default is slow).

## Development

Under `AIRWAY_ENV=local`, `airway server` builds the bundle **in memory**:
`/assets/*` always serves the freshest build, and saving any `.ts/.tsx/.css`
file under `app/assets/js` (vendor excluded) triggers an incremental rebuild
plus a livereload page refresh over the app's WebSocket. There is no HMR
server, no watcher process to start — it is all inside the server binary.

In production the committed `dist/` is embedded via `go:embed`; deploy
remains a single self-contained binary.

## Islands

Embed an island in any templ view:

```go
@assets.Island("counter", map[string]any{"start": 3})
```

This renders:

```html
<div data-island="counter" data-island-id="1"></div>
<script type="application/json" id="island-data-1">{"start":3}</script>
```

The runtime scans `[data-island]` nodes, parses the adjacent JSON as props,
and mounts the component registered for that name. Rules:

- The island name is the file path under `islands/` without extension —
  `islands/admin/chart.tsx` is `"admin/chart"` — and it is case-sensitive.
- Every island file **default-exports** its component.
- Adding an island file needs no registration; the next build picks it up.
- Pages degrade gracefully without JavaScript: the mount point stays empty
  and the props remain inert JSON.

Sources import `react` (aliased onto `preact/compat` at build time), so
React semantics apply — including `forwardRef`, which custom inputs used
with `register()` must use.

## airway-ui

The component library islands build on (`/ui` on any running server shows
everything live):

| Area | Components |
| --- | --- |
| Basics | Button, Input, Textarea, Select, Checkbox, Radio, Field, Spinner, EmptyState |
| Forms | Form (react-hook-form integration) |
| Data | DataTable (TanStack Table: sorting, pagination, row selection, loading/empty states) |
| Feedback | Modal (focus trap, Esc), Toast (`useToast`), Tabs, Pagination |
| Fetching | `apiFetch`, `useApiQuery`, `ApiError`, `setApiErrorHandler` |

The fetch layer speaks `lib/render`'s envelope `{"code":0,"data":…,"message":""}`
(code 0 = success). `setApiErrorHandler(fn)` is the global hook for
business errors such as a 401 redirect.

## Tutorial: from scaffold to a custom island

1. Generate a CRUD resource and start the server:

   ```bash
   airway generate scaffold post title:string
   go generate ./...
   airway js:build
   airway db:migrate
   airway server   # http://127.0.0.1:1900/posts
   ```

2. Scaffold your own island and embed it:

   ```bash
   airway generate island chart
   ```

   ```go
   // in a .templ view
   @assets.Island("chart", map[string]any{"points": []int{4, 8, 15}})
   ```

3. Build it (or just save the file under local dev) and refresh:

   ```bash
   airway js:build
   ```

Edit `app/assets/js/islands/chart.tsx` using the airway-ui components —
that is the whole loop.
