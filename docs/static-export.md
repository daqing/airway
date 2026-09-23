# Static Export (App Pages for CDN)

Airway can export server-rendered app pages as a self-contained static
directory: plain `index.html` files plus the committed frontend bundle, ready
to upload to any static host or CDN. The page HTML is produced by the same
templ components the server renders; the interactive islands keep working,
because the island runtime mounts client-side from the props embedded in each
page (see the [frontend guide](frontend.md)).

This is the app-page counterpart of [static showcase sites](ssg.md): `ssg:build`
exports theme-driven site projects, while `static:build` exports pages of an
Airway application itself. 中文版: [docs/zh-CN/static-export.md](zh-CN/static-export.md).

## Quick start

```bash
go run . static:build          # rebuilds the frontend bundle, renders registered pages, copies assets into ./dist
airway static:serve            # preview on http://127.0.0.1:3000
```

Inside a project, a globally installed `airway` proxies to `go run .`
automatically, so `airway static:build` and `go run . static:build` are
equivalent.

Two inputs feed the export, and each has its own refresh rule:

- **Frontend sources** (`app/assets/js`) are rebuilt automatically before
  every export — no manual `js:build` needed. Without a vendor directory
  (`js:install` not run) the command falls back to the committed bundle with
  a warning; a real build error aborts the export.
- **templ views** (`app/views/**.templ`) are refreshed the same way: when a
  view is newer than its generated file, the command regenerates it and
  re-runs itself, so a single `static:build` exports the edited views. A
  failing regeneration aborts the export instead of shipping stale pages.

## Registering pages

Fresh projects ship with a ready-to-run `export.go` in the project root that
exports the welcome page, so `static:build` works out of the box. Pages are
registered from init functions in the project's root package through
`cmd.SetStaticPages` — the same compile-time registration pattern as plugins,
REPL models, and the ssg site builder:

```go
package main

import (
	"github.com/daqing/airway/app/views/home"
	"github.com/daqing/airway/app/views/posts"
	"github.com/daqing/airway/cmd"
	"github.com/daqing/airway/lib/static"
)

func init() {
	cmd.SetStaticPages(
		static.Page{Slug: "/", Component: home.Index(3)},
		static.Page{Slug: "/about", Component: posts.AboutPage()},
	)
}
```

Each page becomes `<out>/<slug>/index.html` (`/` produces `index.html`,
`/about` produces `about/index.html`). Slugs are normalized to clean absolute
paths with the same rules as `ssg:build`, and duplicates are rejected.

### Scaffolded resources

`airway generate scaffold post title:string` writes the registration for you
in `export_posts.go` — one file per scaffolded resource, so repeated scaffolds
never need to merge into a shared file. The list page is static; detail pages
are enumerated from the database at export time through
`cmd.SetStaticPagesProvider`:

```go
func init() {
	cmd.SetStaticPages(
		static.Page{Slug: "/posts", Component: posts.Index()},
	)

	// Detail pages are enumerated at export time — one per post.
	// AIRWAY_DSN must be configured when running static:build/static:serve.
	cmd.SetStaticPagesProvider(func() ([]static.Page, error) {
		items, err := repo.FindAll[models.Post]()
		if err != nil {
			return nil, fmt.Errorf("enumerate posts: %w", err)
		}

		pages := make([]static.Page, 0, len(items))
		for _, item := range items {
			pages = append(pages, static.Page{
				Slug:      fmt.Sprintf("/posts/%d", item.ID),
				Component: posts.Show(item),
			})
		}
		return pages, nil
	})
}
```

The scaffold also generates the detail page itself (`app/views/posts/show.templ`,
rendered by `GET /posts/:id`): full HTML with the row's content baked in —
that is the page search engines see. Providers run only while
`static:build`/`static:serve` collects pages (never at process start), and
`static:build` connects to the database from `AIRWAY_DSN`/`DSN` when a
provider needs it; projects without database-backed pages export fine without
a DSN.

## How it works

- **Rendering is offline.** A page's `Component` is rendered with
  `context.Background()` — no HTTP request, no database. Whatever data the
  component captures at registration time is what gets exported.
- **Islands survive the export.** `assets.Island(...)` renders an empty mount
  point plus an inline `<script type="application/json">` props block; the
  Preact runtime in `app.js` mounts into those points client-side. There is no
  SSR hydration step, so the static HTML and the mounted islands can never
  disagree.
- **Assets ship next to the pages.** `static:build` copies
  `app/assets/dist/` into `<out>/assets/`, matching the `/assets/app.js?v=…`
  URLs the layout emits (the `?v=` hash comes from the build manifest).
- **The environment is pinned.** The command forces `AIRWAY_ENV=production`
  for the render, so a developer's local `.env` cannot produce dev asset URLs
  (in-memory livereload bundle) that would 404 on a static host.

`static:serve` renders the same pages over HTTP for preview and serves
`app/assets/dist` under `/assets/`, so the preview is fully interactive
without exporting first.

## What exports well — and what does not

| Page kind | Static export | Notes |
| --- | --- | --- |
| Static content pages (landing, docs, marketing) | ✅ | Full HTML, SEO-complete |
| Islands whose props are known at build time | ✅ | Props are baked into the JSON block |
| Pages that query the database per request | ⚠️ | You export a build-time snapshot: enumerate rows at registration time, or rebuild + redeploy when content changes |
| Dynamic routes (`/posts/:id`) | ⚠️ | Register one page per instance (loop in `init()`) |
| POST forms, WebSocket, auth, `useApiQuery` calls | ❌ | Those need a live server; a CDN only hosts the shell |

Rule of thumb: content belongs in the templ layer (it is what search engines
see — island mount points are empty in the HTML), and islands wrap the
interactive parts.

## Deployment

`airway static:build` writes a self-contained directory:

```
dist/
  index.html            # one index.html per registered page
  about/index.html
  assets/               # app/assets/dist copied verbatim
    app.js  app.css  manifest.json  *.map
```

Upload the whole directory to any static host — GitHub Pages, Netlify,
object storage behind a CDN. Two knobs matter:

- **Sub-path deployments**: set `URL_PREFIX` (or `AIRWAY_URL_PREFIX`) when
  building so asset URLs carry the prefix, and upload the directory under
  that path.
- **templ view changes** are picked up automatically: the command
  regenerates stale views and re-runs itself, so no manual
  `templates:compile` step is needed before exporting.

A fresh clone rebuilds everything with `go run . static:build` alone.
