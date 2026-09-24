# OpenAPI API Documentation

Airway generates an [OpenAPI](https://spec.openapis.org) 3.2 document for
your API, so client applications (Vue 3, React, SwiftUI, ...) can be wired up
with standard code-generation tools.

Two ways to consume the document:

- **File**: `airway openapi:generate` writes `./openapi.json` — point your
  generators at it (regenerate whenever routes or declared types change).
- **Live endpoint**: a running server answers `GET /openapi.json` (code
  generators can target a deployed server directly).

```bash
airway openapi:generate                    # writes ./openapi.json
airway openapi:generate --out docs/api.json
```

The output is deterministic (paths, operations and schemas are sorted), so
regenerating after a change keeps diffs clean; the file itself is a local
build artifact (git-ignored). `info.version` comes from the project's
`VERSION` file, and the `servers` entry is derived from the request host and
`URL_PREFIX` on the live endpoint, or from `LISTEN` and `URL_PREFIX`
on the CLI. When the app is deployed behind a reverse proxy with
`URL_PREFIX=/airway`, the document is served at `/airway/openapi.json` and its
`servers` URL carries the prefix.

## What is documented automatically

Every route registered on the Gin engine — including routes mounted by
[plugins](plugin.md) — appears in the document without any extra work:

- `operationId` derived from the handler name (`storage_api.UploadAction`);
- a tag from the first path segment outside `/api/v1`;
- path parameters detected from the route (`:id` / `*key` are normalized to
  `{id}` / `{key}`);
- a default `200` response shaped like the framework envelope (see below).

Routes that carry no JSON contract stay out of the document. The framework
itself excludes `/`, `/ui`, `/assets/*`, `/ws` and `/openapi.json`; adjust the
list in `app/api/openapi_api/doc.go`.

## Declaring richer metadata

An auto-documented route has no request/response schemas. To document them,
add an `openapi.go` file to the API module and declare operations in `init()`
(the same registration pattern as REPL models and plugins):

```go
// app/api/post_api/openapi.go
package post_api

import "github.com/daqing/airway/lib/openapi"

func init() {
	openapi.Get("/api/v1/posts", func(o *openapi.Operation) {
		o.Summary("List posts").Tag("posts").
			Query("page", openapi.Int(), "Page number").
			OK(openapi.List[Post]())
	})

	openapi.Post("/api/v1/posts", func(o *openapi.Operation) {
		o.Summary("Create a post").Tag("posts").
			Body(openapi.Item[CreatePostParams]()).
			OK(openapi.Item[Post]())
	})

	openapi.Put("/api/v1/posts/{id}", func(o *openapi.Operation) {
		o.Summary("Update a post").Tag("posts").
			Path("id", openapi.Int(), "Post id").
			Body(openapi.Item[CreatePostParams]()).
			OK()
	})

	openapi.Delete("/api/v1/posts/{id}", func(o *openapi.Operation) {
		o.Summary("Delete a post").Tag("posts").
			Path("id", openapi.Int(), "Post id").
			OK()
	})
}
```

A declaration is matched against the registered route by method + path (gin
`:id` / `*key` styles are accepted and normalized). Anything the declaration
sets overrides the auto-documented defaults; `airway openapi:generate` warns
about declarations that match no registered route.

### Response envelope

`render.OK` and `render.Error` both answer with HTTP 200 and the
`{"code", "data", "message"}` envelope, so `o.OK(schema)` documents the 200
response as `oneOf` the success envelope (with your schema as `data`) and the
shared `Error` component — which is exactly what clients must handle.

Endpoints that bypass the envelope (raw `c.JSON`, file downloads) declare
their responses explicitly:

```go
o.Respond(200, "application/octet-stream", openapi.File()).
	Description("The file contents")
o.Respond(404, "application/json",
	openapi.Obj(map[string]*openapi.Schema{"error": openapi.Str()}))
```

### Schema helpers

| Helper | Shape |
| --- | --- |
| `openapi.Item[T]()` | single value of `T`; named structs become `$ref` components inferred from `json` tags |
| `openapi.List[T]()` | JSON array of `T` |
| `openapi.Obj(props)` | object from a property map |
| `openapi.Str()` / `Int()` / `Num()` / `Bool()` / `Any()` | scalars |
| `openapi.File()` | binary string (`format: binary`) |

Inference rules: `json` tags name the properties (`-` excluded, `omitempty`
makes the field optional), `time.Time` becomes `string` / `date-time`,
pointers become nullable types, and nested named structs become referenced
components. Re-run `airway openapi:generate` after changing a struct.

### Operation options

`Summary`, `Description`, `Tag`, `ID` (operationId override), `Deprecated`,
`Query`, `Path`, `Body` (JSON), `Form` (form-urlencoded), `FormUpload`
(multipart, e.g. storage uploads), `Respond` (status + content type), and
`Security` (per-operation requirement).

## Document-level settings

Title, exclusions, tag metadata and security schemes are declared once in
`app/api/openapi_api/doc.go`:

```go
func init() {
	openapi.DescribeDoc(func(d *openapi.Document) {
		d.Title("My App API").
			Exclude("/legacy/*").
			Tag("posts", "Blog management").
			SecurityScheme("bearer", openapi.Bearer("JWT"))
	})
}
```

Operations then opt in with `o.Security("bearer")`. With no declared security
schemes the document carries an empty root `security` list, marking the API
as public.

## Scaffolding

`airway generate api` and `airway generate scaffold` emit a starter
`openapi.go` next to the routes, so new modules are documented from birth —
update the declarations as you flesh out the handlers.

## Generating clients

Point any OpenAPI 3.x-capable generator at `openapi.json` (or the live
endpoint URL):

**Vue 3 / React (TypeScript)** — [openapi-typescript](https://openapi-ts.dev)
plus [openapi-fetch](https://openapi-fetch.dev):

```bash
npx openapi-typescript ./openapi.json -o src/api/schema.d.ts
```

**Alternative:** [orval](https://orval.dev) generates typed hooks (react-query)
or axios clients from the same document.

**SwiftUI** — Apple's
[swift-openapi-generator](https://github.com/apple/swift-openapi-generator)
consumes 3.2 documents directly: add the generator build plugin and
[OpenAPIRuntime](https://swiftpackageindex.com/swift-server/openapi-runtime)
to your Xcode project or Swift package, point it at `openapi.json`, and it
emits typed client methods for every operation. The generated client reads
the `servers` URL from the document, so the `URL_PREFIX` deployment setting
is honored automatically.
