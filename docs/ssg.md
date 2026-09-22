# Static Showcase Sites

Airway can generate static showcase websites — company homepages, product
landings, portfolios — as plain HTML directories deployable to any static
host. Where a classic static site generator renders template files, Airway
renders **Go code**: pages are declared in a Go file, themes are Go modules
supplying [templ](https://templ.guide) components and embedded assets, and
the export is a self-contained `dist/` directory.

A showcase site needs no database, no Node toolchain, and no running server.

> Background and design record: [SSG.md](../SSG.md) (中文版:[SSG.zh-CN.md](../SSG.zh-CN.md)).

## Quick start

```bash
airway ssg:new mysite           # scaffold a site project (or an absolute path)
cd mysite
airway ssg:build                # export the static site into dist/
airway ssg:serve                # preview on http://127.0.0.1:3000
```

`ssg:new` creates a minimal Go project: a `main.go` host, a `ssg.go` with
the site definition, and a README. It accepts the same arguments as
`airway new` (a module path, or an absolute directory); `--local[=path]`
points both the framework and the corporate theme at a local airway
checkout, which is how framework development stays productive.

## Defining the site

`ssg.go` in the project root is the single source of truth. It registers a
builder with `cmd.SetSSGBuilder`, which `airway ssg:build` and
`airway ssg:serve` run on — the same compile-time registration pattern
plugins and REPL models use:

```go
func init() {
	cmd.SetSSGBuilder(func() (*ssg.Site, error) {
		s := site.New(ssg.Meta{
			Title:       "Acme Inc",
			Description: "We build ships.",
		})

		s.Use(corporate.Theme)

		s.Page("/", "Home", corporate.Home{
			Heading: "We build ships",
			Tagline: "Family-owned since 1951.",
			Features: []corporate.Feature{
				{Title: "Design", Body: "Hull-first thinking."},
			},
		})

		s.Page("/about", "About", corporate.Content{
			Lead:       "Family-owned since 1951.",
			Paragraphs: []string{"We operate out of Hamburg."},
		})

		return s, nil
	})
}
```

Each `Page(slug, title, data)` call is one output page. The slug is
normalized to a clean absolute path and decides the export location: `/`
becomes `dist/index.html`, `/about` becomes `dist/about/index.html`. The
theme's `Assets()` are copied into `dist/assets/`.

Inside a site project a globally installed `airway` proxies to `go run .`
automatically, so `airway ssg:build` and `go run . ssg:build` are
equivalent.

## Commands

```bash
airway ssg:new [--local[=path]] <module-path | directory>   # scaffold a site project
airway ssg:build [--out dist]                               # export static HTML
airway ssg:serve [--addr 127.0.0.1:3000]                    # local preview server
airway theme:new [--local[=path]] <module-path | path>       # scaffold a new theme module
airway theme:install <module | /path/to/theme>               # wire a theme into a site project
```

`ssg:serve` renders pages dynamically over HTTP (assets stream from the
theme's embedded filesystem) — handy for previewing; the exported `dist/`
from `ssg:build` is what you deploy.

## The corporate theme

`github.com/daqing/airway/themes/corporate` ships with the framework: a
responsive layout with header navigation (built from the site's pages, with
the current page highlighted), a hero + feature grid landing page, plain
text pages, and a footer. `Site.Render` dispatches on the page data type:

| Data type | Renders |
|---|---|
| `corporate.Home` | landing page: hero heading, tagline, feature grid |
| `corporate.Content` | text page: lead plus paragraphs |
| `templ.Component` | a custom body inside the shared layout |
| anything else (incl. `nil`) | header and footer only |

## Installing themes

Third-party themes install into a site project with `airway theme:install`
— from a published module or a local checkout:

```bash
airway theme:install github.com/daqing/airway-terminal-theme
airway theme:install github.com/example/theme@v1.2.3
airway theme:install ~/src/airway-terminal-theme   # local checkout: adds a replace
```

The command wires the theme into the project's go.mod (`require` for
published modules, `replace` + `require` for a local directory, which must
be a Go module requiring `github.com/daqing/airway`), resolves the theme's
package name, and prints the two lines that enable it in `ssg.go`. Nothing
is copied into the project — themes are consumed as Go module dependencies.

## Writing a theme

Scaffold a theme module with `airway theme:new` — it derives the Go package
name from the module name (`github.com/me/airway-sunset-theme` becomes
package `sunset`), writes the `Theme` implementation, templ views, starter
CSS, and tests, pins the templ generation to the framework's, generates the
templ views, and initializes a git repository:

```bash
airway theme:new github.com/me/airway-sunset-theme
airway theme:new --local sunset   # develop against the airway checkout in $PWD
```

A theme is any Go type implementing `lib/ssg.Theme`:

```go
type Theme interface {
	Name() string
	Render(ctx *site.Context) templ.Component
	Assets() fs.FS
}
```

- `Render` receives the site metadata (`ctx.Meta`), the current page
  (`ctx.Page`), and every registered page (`ctx.Pages`) for navigation. It
  returns a full HTML document; ship the CSS through `Assets()`.
- `Assets` returns an `fs.FS` copied into the site's `assets/` directory and
  served under `/assets/` — embed it with `go:embed` (re-root the FS with
  `fs.Sub` if the embedded files live in a subdirectory).

Themes live in their own Go modules (see `themes/corporate` in the
framework repository for a complete example, including tests). A site
selects one with `s.Use(mytheme.Theme)`.

## Deployment

`airway ssg:build` writes a self-contained static directory: one
`index.html` per page plus `assets/`. Upload it to any static host — GitHub
Pages, Netlify, object storage behind a CDN. A fresh clone rebuilds it with
`go run . ssg:build` alone.
