package openapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Operation describes one API operation. The method/path pair identifies the
// registered Gin route the declaration merges into; path uses OpenAPI
// parameter style ("/posts/{id}") — gin-style ":id" and "*key" are accepted
// and normalized.
type Operation struct {
	method      string
	path        string
	tags        []string
	summary     string
	description string
	operationID string
	deprecated  bool
	parameters  []Parameter
	requestBody *RequestBody
	responses   map[string]*Response
	security    []map[string][]string
}

// MarshalJSON emits the OpenAPI operation object; responses is mandatory in
// the spec and always present on merged operations.
func (o *Operation) MarshalJSON() ([]byte, error) {
	type operationOut struct {
		Tags        []string              `json:"tags,omitempty"`
		Summary     string                `json:"summary,omitempty"`
		Description string                `json:"description,omitempty"`
		OperationID string                `json:"operationId,omitempty"`
		Deprecated  bool                  `json:"deprecated,omitempty"`
		Parameters  []Parameter           `json:"parameters,omitempty"`
		RequestBody *RequestBody          `json:"requestBody,omitempty"`
		Responses   map[string]*Response  `json:"responses"`
		Security    []map[string][]string `json:"security,omitempty"`
	}

	return json.Marshal(operationOut{
		Tags:        o.tags,
		Summary:     o.summary,
		Description: o.description,
		OperationID: o.operationID,
		Deprecated:  o.deprecated,
		Parameters:  o.parameters,
		RequestBody: o.requestBody,
		Responses:   o.responses,
		Security:    o.security,
	})
}

// Parameter describes a path, query, header or cookie parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody describes a request payload.
type RequestBody struct {
	Description string           `json:"description,omitempty"`
	Required    bool             `json:"required,omitempty"`
	Content     map[string]Media `json:"content,omitempty"`
}

// Response describes one named response.
type Response struct {
	description string
	content     map[string]Media
}

// MarshalJSON emits the OpenAPI response object; description is mandatory.
func (r *Response) MarshalJSON() ([]byte, error) {
	type responseOut struct {
		Description string           `json:"description"`
		Content     map[string]Media `json:"content,omitempty"`
	}

	return json.Marshal(responseOut{
		Description: r.description,
		Content:     r.content,
	})
}

// Description sets the response description; useful right after Respond.
func (r *Response) Description(desc string) *Response {
	r.description = desc
	return r
}

// Media carries the schema for one content type.
type Media struct {
	Schema *Schema `json:"schema,omitempty"`
}

// Get declares a GET operation; call from a module's openapi.go init().
func Get(path string, fn func(o *Operation)) { declare(http.MethodGet, path, fn) }

// Post declares a POST operation.
func Post(path string, fn func(o *Operation)) { declare(http.MethodPost, path, fn) }

// Put declares a PUT operation.
func Put(path string, fn func(o *Operation)) { declare(http.MethodPut, path, fn) }

// Patch declares a PATCH operation.
func Patch(path string, fn func(o *Operation)) { declare(http.MethodPatch, path, fn) }

// Delete declares a DELETE operation.
func Delete(path string, fn func(o *Operation)) { declare(http.MethodDelete, path, fn) }

var (
	declMu   sync.Mutex
	declared = map[string]*Operation{} // "METHOD path" -> declaration
	docDecl  *Document
)

func declare(method, path string, fn func(o *Operation)) {
	op := &Operation{method: strings.ToUpper(method), path: normalizePath(path)}
	if fn != nil {
		fn(op)
	}

	declMu.Lock()
	defer declMu.Unlock()

	key := op.method + " " + op.path
	if _, dup := declared[key]; dup {
		panic("openapi: duplicate declaration for " + key)
	}
	declared[key] = op
}

func (o *Operation) Summary(s string) *Operation {
	o.summary = s
	return o
}

func (o *Operation) Description(s string) *Operation {
	o.description = s
	return o
}

// ID overrides the operationId (by default it is the handler's short name,
// e.g. "storage_api.UploadAction").
func (o *Operation) ID(id string) *Operation {
	o.operationID = id
	return o
}

func (o *Operation) Tag(names ...string) *Operation {
	o.tags = append(o.tags, names...)
	return o
}

func (o *Operation) Deprecated() *Operation {
	o.deprecated = true
	return o
}

// Query documents a query parameter.
func (o *Operation) Query(name string, schema *Schema, desc string) *Operation {
	o.parameters = append(o.parameters, Parameter{
		Name: name, In: "query", Description: desc, Schema: schema,
	})
	return o
}

// Path documents a path parameter ({id} in the declared path).
func (o *Operation) Path(name string, schema *Schema, desc string) *Operation {
	o.parameters = append(o.parameters, Parameter{
		Name: name, In: "path", Required: true, Description: desc, Schema: schema,
	})
	return o
}

// Body documents a JSON request payload.
func (o *Operation) Body(schema *Schema) *Operation {
	o.requestBody = &RequestBody{
		Required: true,
		Content:  map[string]Media{"application/json": {Schema: schema}},
	}
	return o
}

// Form documents an application/x-www-form-urlencoded payload.
func (o *Operation) Form(fields map[string]*Schema) *Operation {
	o.requestBody = formRequestBody("application/x-www-form-urlencoded", fields)
	return o
}

// FormUpload documents a multipart/form-data payload; file fields use
// openapi.File().
func (o *Operation) FormUpload(fields map[string]*Schema) *Operation {
	o.requestBody = formRequestBody("multipart/form-data", fields)
	return o
}

func formRequestBody(contentType string, fields map[string]*Schema) *RequestBody {
	return &RequestBody{
		Required: true,
		Content: map[string]Media{
			contentType: {Schema: Obj(fields)},
		},
	}
}

// OK documents a 200 response carrying the framework JSON envelope with the
// given data schema (render.OK); errors are covered by the shared Error
// component (render.Error also answers with HTTP 200). Without a schema the
// data property accepts any value (render.Empty).
func (o *Operation) OK(data ...*Schema) *Operation {
	var payload *Schema
	if len(data) > 0 {
		payload = data[0]
	}
	o.Respond(200, "application/json", envelopeSchema(payload)).
		Description("Success or in-band error envelope (render.OK / render.Error)")
	return o
}

// Respond documents one response status; chain .Description() to label it.
func (o *Operation) Respond(status int, contentType string, schema *Schema) *Response {
	if o.responses == nil {
		o.responses = map[string]*Response{}
	}

	resp := &Response{content: map[string]Media{contentType: {Schema: schema}}}
	o.responses[strconv.Itoa(status)] = resp
	return resp
}

// Security attaches a named security requirement declared via
// Document.SecurityScheme.
func (o *Operation) Security(name string) *Operation {
	o.security = append(o.security, map[string][]string{name: {}})
	return o
}

// SecurityScheme declares how clients authenticate; see Document.SecurityScheme.
type SecurityScheme struct {
	Type         string `json:"type"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}

// Bearer returns an HTTP bearer security scheme (e.g. Bearer("JWT")).
func Bearer(format string) SecurityScheme {
	return SecurityScheme{Type: "http", Scheme: "bearer", BearerFormat: format}
}

// Basic returns an HTTP basic security scheme.
func Basic() SecurityScheme {
	return SecurityScheme{Type: "http", Scheme: "basic"}
}

// APIKey returns an API key security scheme read from a header, query
// parameter or cookie (in: "header", "query" or "cookie").
func APIKey(name, in string) SecurityScheme {
	return SecurityScheme{Type: "apiKey", Name: name, In: in}
}

// Document holds document-level settings applied by DescribeDoc.
type Document struct {
	title       string
	summary     string
	description string
	version     string

	exclusions []string
	security   map[string]SecurityScheme
	tags       []TagDecl
}

// TagDecl is one entry of the document's tag list.
type TagDecl struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// DescribeDoc applies document-level settings; call from the app's own
// openapi module (see app/api/openapi_api/doc.go).
func DescribeDoc(fn func(d *Document)) {
	declMu.Lock()
	defer declMu.Unlock()

	if docDecl == nil {
		docDecl = &Document{}
	}
	fn(docDecl)
}

// Title sets the document title (info.title).
func (d *Document) Title(t string) *Document { d.title = t; return d }

// Summary sets info.summary (OpenAPI 3.1+).
func (d *Document) Summary(s string) *Document { d.summary = s; return d }

// Description sets info.description.
func (d *Document) Description(s string) *Document { d.description = s; return d }

// Version overrides info.version (by default the project's VERSION file).
func (d *Document) Version(v string) *Document { d.version = v; return d }

// Exclude hides routes from the generated document. A pattern matches the
// normalized path exactly; a trailing "*" turns it into a prefix match
// ("/assets/*" hides everything under /assets/).
func (d *Document) Exclude(patterns ...string) *Document {
	d.exclusions = append(d.exclusions, patterns...)
	return d
}

// SecurityScheme registers a named security scheme.
func (d *Document) SecurityScheme(name string, scheme SecurityScheme) *Document {
	if d.security == nil {
		d.security = map[string]SecurityScheme{}
	}
	d.security[name] = scheme
	return d
}

// Tag registers tag metadata shown in the document's tag list.
func (d *Document) Tag(name, description string) *Document {
	d.tags = append(d.tags, TagDecl{Name: name, Description: description})
	return d
}

// envelopeSchema wraps data in the render.OK / render.Error response shape:
// oneOf the success envelope and the shared Error component (both answer
// with HTTP 200).
func envelopeSchema(data *Schema) *Schema {
	if data == nil {
		data = Any()
	}

	registerComponent("Error", &Schema{
		Types: TypeList{"object"},
		Properties: map[string]*Schema{
			"code":    Int(),
			"data":    Any(),
			"message": Str(),
		},
		Required: []string{"code", "message"},
	})

	success := &Schema{
		Types: TypeList{"object"},
		Properties: map[string]*Schema{
			"code":    Int(),
			"data":    data,
			"message": Str(),
		},
		Required: []string{"code", "data", "message"},
	}

	return &Schema{OneOf: []*Schema{
		success,
		{Ref: "#/components/schemas/Error"},
	}}
}
