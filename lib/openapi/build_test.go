package openapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func openapiTestIndex(c *gin.Context)  {}
func openapiTestCreate(c *gin.Context) {}
func openapiTestUpdate(c *gin.Context) {}

func mustBuild(t *testing.T, r *gin.Engine, opts BuildOptions) (map[string]any, []byte) {
	t.Helper()

	body, err := Build(r, opts)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return doc, body
}

func dig(t *testing.T, doc map[string]any, path string) any {
	t.Helper()

	current := any(doc)
	for _, part := range strings.Split(path, ".") {
		switch container := current.(type) {
		case map[string]any:
			value, ok := container[part]
			if !ok {
				t.Fatalf("missing key %q while digging %q", part, path)
			}
			current = value
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(container) {
				t.Fatalf("bad index %q while digging %q", part, path)
			}
			current = container[index]
		default:
			t.Fatalf("cannot dig %q: %q is not a container", path, part)
		}
	}
	return current
}

func TestBuildDefaultsForUndeclaredRoute(t *testing.T) {
	resetRegistry()

	r := gin.New()
	r.GET("/api/v1/posts", openapiTestIndex)

	doc, _ := mustBuild(t, r, BuildOptions{})

	if got := dig(t, doc, "openapi"); got != "3.2.0" {
		t.Fatalf("expected OpenAPI 3.2.0, got %v", got)
	}
	if got := dig(t, doc, "paths./api/v1/posts.get.operationId"); got != "openapi.openapiTestIndex" {
		t.Fatalf("unexpected operationId: %v", got)
	}
	if got := dig(t, doc, "paths./api/v1/posts.get.tags"); marshalCompact(t, got) != `["posts"]` {
		t.Fatalf("unexpected tags: %v", got)
	}

	// Default response: the framework envelope as oneOf(success, Error).
	oneOf := dig(t, doc, "paths./api/v1/posts.get.responses.200.content.application/json.schema.oneOf").([]any)
	if len(oneOf) != 2 {
		t.Fatalf("expected success+error oneOf, got %v", oneOf)
	}
	if ref := dig(t, doc, "paths./api/v1/posts.get.responses.200.content.application/json.schema.oneOf.1.$ref"); ref != "#/components/schemas/Error" {
		t.Fatalf("unexpected error ref: %v", ref)
	}
	if dig(t, doc, "components.schemas.Error") == nil {
		t.Fatal("missing Error component")
	}
}

func TestBuildMergesDeclaration(t *testing.T) {
	resetRegistry()

	Get("/api/v1/posts", func(o *Operation) {
		o.Summary("List posts").Tag("blog").
			Query("page", Int(), "Page number").
			OK(List[openapiTestPost]())
	})

	r := gin.New()
	r.GET("/api/v1/posts", openapiTestIndex)

	doc, _ := mustBuild(t, r, BuildOptions{})

	if got := dig(t, doc, "paths./api/v1/posts.get.summary"); got != "List posts" {
		t.Fatalf("unexpected summary: %v", got)
	}
	if got := dig(t, doc, "paths./api/v1/posts.get.tags"); marshalCompact(t, got) != `["blog"]` {
		t.Fatalf("unexpected tags: %v", got)
	}
	if got := dig(t, doc, "paths./api/v1/posts.get.operationId"); got != "openapi.openapiTestIndex" {
		t.Fatalf("declaration without ID must keep the handler operationId, got %v", got)
	}
	if got := dig(t, doc, "paths./api/v1/posts.get.parameters.0.name"); got != "page" {
		t.Fatalf("unexpected query parameter: %v", got)
	}
	if got := dig(t, doc, "paths./api/v1/posts.get.responses.200.content.application/json.schema.oneOf.0.properties.data.items.$ref"); got != "#/components/schemas/openapiTestPost" {
		t.Fatalf("unexpected data ref: %v", got)
	}
	if dig(t, doc, "components.schemas.openapiTestPost") == nil {
		t.Fatal("missing openapiTestPost component")
	}
}

func TestBuildNormalizesGinPathAndShadowedParams(t *testing.T) {
	resetRegistry()

	r := gin.New()
	r.PUT("/api/v1/posts/:id", openapiTestUpdate)

	doc, _ := mustBuild(t, r, BuildOptions{})

	if dig(t, doc, "paths./api/v1/posts/{id}.put") == nil {
		t.Fatal("expected normalized path /api/v1/posts/{id}")
	}
	if got := dig(t, doc, "paths./api/v1/posts/{id}.put.parameters.0"); marshalCompact(t, got) != `{"in":"path","name":"id","required":true,"schema":{"type":"string"}}` {
		t.Fatalf("unexpected auto path parameter: %v", got)
	}

	resetRegistry()
	Put("/api/v1/posts/:id", func(o *Operation) {
		o.Path("id", Int(), "Post ID").OK()
	})

	doc, _ = mustBuild(t, r, BuildOptions{})

	params := dig(t, doc, "paths./api/v1/posts/{id}.put.parameters").([]any)
	if len(params) != 1 {
		t.Fatalf("declared parameter must shadow the auto-detected one, got %v", params)
	}
	if got := dig(t, doc, "paths./api/v1/posts/{id}.put.parameters.0.schema.type"); got != "integer" {
		t.Fatalf("unexpected declared parameter type: %v", got)
	}
}

func TestBuildExcludesRoutes(t *testing.T) {
	resetRegistry()

	DescribeDoc(func(d *Document) {
		d.Exclude("/api/v1/posts", "/assets/*")
	})

	r := gin.New()
	r.GET("/api/v1/posts", openapiTestIndex)
	r.GET("/assets/app.js", openapiTestIndex)
	r.GET("/health", openapiTestIndex)

	doc, _ := mustBuild(t, r, BuildOptions{})

	paths := dig(t, doc, "paths").(map[string]any)
	if len(paths) != 1 {
		t.Fatalf("expected only /health to remain, got %v", paths)
	}
	if dig(t, doc, "paths./health.get") == nil {
		t.Fatal("expected /health to be documented")
	}
}

func TestBuildDeterministicOutput(t *testing.T) {
	resetRegistry()

	Post("/api/v1/posts", func(o *Operation) {
		o.Summary("Create post").Tag("posts").
			Body(Item[openapiTestPost]()).
			OK(Item[openapiTestPost]())
	})

	r := gin.New()
	r.GET("/api/v1/posts", openapiTestIndex)
	r.POST("/api/v1/posts", openapiTestCreate)
	r.DELETE("/api/v1/posts/{id}", openapiTestUpdate)

	_, first := mustBuild(t, r, BuildOptions{})
	_, second := mustBuild(t, r, BuildOptions{})

	if string(first) != string(second) {
		t.Fatalf("build output must be deterministic:\n%s\n---\n%s", first, second)
	}
}

func TestBuildWarnsOnUnmatchedDeclaration(t *testing.T) {
	resetRegistry()

	Delete("/api/v1/posts/{id}", func(o *Operation) {})

	r := gin.New()
	r.GET("/api/v1/posts", openapiTestIndex)

	var warnings []string
	_, _ = mustBuild(t, r, BuildOptions{Warn: func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}})

	if len(warnings) != 1 || !strings.Contains(warnings[0], "/api/v1/posts/{id}") {
		t.Fatalf("expected a warning for the unmatched declaration, got %v", warnings)
	}
}

func TestBuildInfoAndServerOptions(t *testing.T) {
	resetRegistry()

	r := gin.New()
	r.GET("/health", openapiTestIndex)

	doc, _ := mustBuild(t, r, BuildOptions{
		Title:   "My App",
		Version: "v1.2.3",
		Server:  "http://localhost:1900",
	})

	if got := dig(t, doc, "info.title"); got != "My App" {
		t.Fatalf("unexpected title: %v", got)
	}
	if got := dig(t, doc, "info.version"); got != "v1.2.3" {
		t.Fatalf("unexpected version: %v", got)
	}
	if got := dig(t, doc, "servers.0.url"); got != "http://localhost:1900" {
		t.Fatalf("unexpected server: %v", got)
	}
}

func TestBuildSecurityScheme(t *testing.T) {
	resetRegistry()

	DescribeDoc(func(d *Document) {
		d.SecurityScheme("bearer", Bearer("JWT"))
	})
	Get("/api/v1/posts", func(o *Operation) {
		o.Summary("List posts").Security("bearer").OK()
	})

	r := gin.New()
	r.GET("/api/v1/posts", openapiTestIndex)

	doc, _ := mustBuild(t, r, BuildOptions{})

	if got := dig(t, doc, "components.securitySchemes.bearer.scheme"); got != "bearer" {
		t.Fatalf("unexpected scheme: %v", got)
	}
	if got := dig(t, doc, "paths./api/v1/posts.get.security.0.bearer"); got == nil {
		t.Fatal("missing operation security requirement")
	}
}

func TestExcludedPathPatterns(t *testing.T) {
	cases := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"/", "/", true},
		{"/ui", "/ui", true},
		{"/assets/app.js", "/assets/*", true},
		{"/assets/*path", "/assets/*path", true},
		{"/api/v1/storage", "/api/v1/*", true},
		{"/health", "/assets/*", false},
	}

	for _, c := range cases {
		if got := excludedPath(c.path, []string{c.pattern}); got != c.want {
			t.Fatalf("excludedPath(%q, %q) = %v, want %v", c.path, c.pattern, got, c.want)
		}
	}
}
