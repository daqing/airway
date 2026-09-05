# Airway Views Guide (templ)

Server-rendered HTML pages in Airway are written with
[templ](https://templ.guide/). A `.templ` file is a normal-looking HTML file
with Go expressions sprinkled in; at build time it is **compiled into a plain
Go function** that returns a typed `templ.Component`. There is no template
parsing at request time — pages are just fast Go code.

This guide shows where views live, how to write one, and how an action renders
it.

## Where views live

Views live under `app/views/`, one folder per API module. The folder name is
the API module name with the `_api` suffix removed:

| API module | View folder | View files |
| --- | --- | --- |
| `home_api` | `app/views/home/` | `index.templ` |
| `post_api` | `app/views/post/` | `show.templ`, `index.templ`, … |
| (shared) | `app/views/layouts/` | `base.templ` |

Each folder is its own Go package, and its name is the folder name. A package
may hold any number of view files.

### Generated files are committed

The generator turns each `*.templ` file into a sibling `*_templ.go` file (for
example `app/views/home/index.templ` → `app/views/home/index_templ.go`). Those
generated files **are committed**, so `go build` and `go test` never need the
templ CLI installed (templ is pinned as a Go tool in `go.mod`). After you edit
a `.templ` file, regenerate and keep the output in sync:

```bash
go generate ./...   # or: just generate
```

## Anatomy of a view

`app/views/home/index.templ` (the landing page) is the smallest useful example:

```templ
package home

import "github.com/daqing/airway/app/views/layouts"

// Index renders the landing page (GET /).
templ Index() {
	@layouts.Base("Airway") {
		<h1>Hello, Airway!</h1>
		<p>This page is rendered with the templ template engine.</p>
	}
}
```

`templ Index()` is a component: a named template, effectively a pure function
from its parameters to HTML. The generated Go code exports `func Index()
templ.Component`, which is what an action calls.

## The shared layout

`app/views/layouts/base.templ` provides the HTML document shell (doctype, head,
title, body). It declares a `children` slot where the page's own content is
injected:

```templ
package layouts

templ Base(title string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="utf-8"/>
			<title>{ title }</title>
		</head>
		<body>
			{ children... }
		</body>
	</html>
}
```

A page passes its markup to the layout by placing it inside the call's braces,
as shown above. Put shared markup — navigation, footers, scripts — in
`layouts/` and wrap every page in it so you don't repeat the shell.

## Rendering a view from an action

Use the `lib/render` HTML helper. This is the counterpart of the JSON helpers
(`render.OK`, `render.Error`) for page responses:

```go
package home_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/views/home"
	"github.com/daqing/airway/lib/render"
)

func IndexAction(c *gin.Context) {
	render.HTML(c, home.Index())
}
```

Available helpers:

- `render.HTML(c, component)` — sends the page with status `200` and
  `Content-Type: text/html`.
- `render.HTMLStatus(c, status, component)` — same, with an explicit status
  code (for example `http.StatusNotFound`).

The component is rendered into a buffer first, so a render failure surfaces as
a `500` instead of a half-written response.

## Passing data to a view

Components take whatever data they need as parameters. A page for a single
record might look like:

```templ
package post

import "github.com/daqing/airway/app/views/layouts"

type Post struct {
	Title     string
	Body      string
	Published bool
	Tags      []string
}

templ Show(post Post) {
	@layouts.Base(post.Title) {
		<h1>{ post.Title }</h1>
		<article>{ post.Body }</article>
	}
}
```

Types referenced by a view can be declared in the same `.templ` file's header
(any Go code there is emitted into the generated file). The action builds the
argument from a model or service result:

```go
func ShowAction(c *gin.Context) {
	post, err := services.FindPostByID(...)   // whatever your service layer returns
	if err != nil {
		render.Error(c, err)
		return
	}

	render.HTML(c, post_view.Show(post))
}
```

Keep components **pure**: pass data in through parameters; don't query the
database or touch the request from inside a template. Handlers gather data and
render; templates only format it.

### Interpolation, control flow, and raw HTML

Anything inside `{ }` is a Go expression. Values are HTML-escaped by default,
so user input can't inject markup. Control flow is plain Go — `if`, `else`,
`for`, and `switch` blocks with Go braces:

```templ
<h1>{ post.Title }</h1>

if post.Published {
	<p>Published</p>
}

<ul>
	for _, tag := range post.Tags {
		<li><a href={ "/tags/" + tag }>{ tag }</a></li>
	}
</ul>
```

If you genuinely need to emit trusted HTML (for example a render of Markdown
that was already sanitised), wrap it with `templ.Raw`:

```templ
<article>{ templ.Raw(post.HTMLBody) }</article>
```

Prefer the escaped `{ expr }` form; reach for `templ.Raw` only for content you
trust.

## Practical example: adding a page

Let's add an HTML `show` page to a `post_api` module.

1. Create `app/views/post/show.templ` (from the example above).
2. Regenerate the Go code:

   ```bash
   go generate ./...
   ```

   This creates `app/views/post/show_templ.go` with `func Show(...)
   templ.Component`.

3. Render it from the action (as in the `ShowAction` snippet above).
4. Wire the route in `config/routes.go` or the module's own `routes.go`.

## Workflow notes

- **Regenerate after every edit.** Edit the `.templ` file, then run
  `go generate ./...` (or `just generate`). Commit the generated `*_templ.go`
  alongside the `.templ` source.
- **Errors show up at compile time.** Because templates compile to Go,
  typos and type mismatches fail the build — not the request.
- **Editor support.** templ ships an LSP (`go tool templ lsp`) and IDE
  extensions; use `go tool templ fmt` to format `.templ` files, or `go tool
  templ fmt -fail .` in CI to enforce it.
- **Security.** String interpolation is escaped by default; build URLs and
  attributes with expressions rather than string-concatenating HTML. Use
  `templ.Raw` only for content you already trust.

## Further reading

- The built-in example: `app/views/home/` + `app/api/home_api/index_action.go`
  (`GET /`).
- The render helper: `lib/render/html.go`.
- The [templ documentation](https://templ.guide/) for the full template syntax.
