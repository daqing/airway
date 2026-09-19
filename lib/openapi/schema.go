// Package openapi generates an OpenAPI 3.2 document for the routes
// registered on the Gin engine.
//
// Every route is documented automatically with the framework's JSON envelope
// ({"code", "data", "message"}); API modules enrich their operations by
// declaring metadata from an openapi.go file (see docs/openapi.md).
package openapi

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Schema is the subset of JSON Schema (2020-12) used by the generator.
// Nil fields are omitted from the emitted document.
type Schema struct {
	Ref             string             `json:"$ref,omitempty"`
	Description     string             `json:"description,omitempty"`
	Types           TypeList           `json:"type,omitempty"`
	Format          string             `json:"format,omitempty"`
	Properties      map[string]*Schema `json:"properties,omitempty"`
	Required        []string           `json:"required,omitempty"`
	Items           *Schema            `json:"items,omitempty"`
	AdditionalProps *Schema            `json:"additionalProperties,omitempty"`
	AnyOf           []*Schema          `json:"anyOf,omitempty"`
	OneOf           []*Schema          `json:"oneOf,omitempty"`
}

// TypeList marshals a single type as a plain string and a nullable type as
// the 2020-12 array form, e.g. "type": ["string", "null"].
type TypeList []string

func (t TypeList) MarshalJSON() ([]byte, error) {
	if len(t) == 1 {
		return json.Marshal(t[0])
	}
	return json.Marshal([]string(t))
}

// Str describes a JSON string.
func Str() *Schema { return &Schema{Types: TypeList{"string"}} }

// Int describes a JSON integer.
func Int() *Schema { return &Schema{Types: TypeList{"integer"}} }

// Num describes a JSON number.
func Num() *Schema { return &Schema{Types: TypeList{"number"}} }

// Bool describes a JSON boolean.
func Bool() *Schema { return &Schema{Types: TypeList{"boolean"}} }

// Any describes any JSON value (an empty schema).
func Any() *Schema { return &Schema{} }

// File describes a binary value (a multipart file field or an octet-stream body).
func File() *Schema { return &Schema{Types: TypeList{"string"}, Format: "binary"} }

// Obj describes a JSON object from its properties; every property is optional.
func Obj(props map[string]*Schema) *Schema {
	return &Schema{Types: TypeList{"object"}, Properties: props}
}

// Item describes a single value of T; named structs become a $ref into
// components/schemas, inferred from the struct's json tags.
func Item[T any]() *Schema {
	return inferType(reflect.TypeOf((*T)(nil)).Elem())
}

// List describes a JSON array of T.
func List[T any]() *Schema {
	return &Schema{Types: TypeList{"array"}, Items: Item[T]()}
}

// Model describes the type of v; a convenience form of Item for call sites
// that already hold a zero value.
func Model(v any) *Schema {
	t := reflect.TypeOf(v)
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return inferType(t)
}

var (
	timeType = reflect.TypeOf(time.Time{})

	compMu      sync.Mutex
	compSchemas = map[string]*Schema{}
	compTypes   = map[string]reflect.Type{}
)

// inferType maps a Go type to a JSON Schema. Named structs (except time.Time)
// are registered as reusable components and referenced by $ref.
func inferType(t reflect.Type) *Schema {
	if t == nil {
		return Any()
	}

	if t == timeType {
		return &Schema{Types: TypeList{"string"}, Format: "date-time"}
	}

	switch t.Kind() {
	case reflect.String:
		return Str()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return Int()
	case reflect.Int64, reflect.Uint64:
		return &Schema{Types: TypeList{"integer"}, Format: "int64"}
	case reflect.Float32, reflect.Float64:
		return Num()
	case reflect.Bool:
		return Bool()
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return &Schema{Types: TypeList{"string"}, Format: "byte"}
		}
		return &Schema{Types: TypeList{"array"}, Items: inferType(t.Elem())}
	case reflect.Array:
		return &Schema{Types: TypeList{"array"}, Items: inferType(t.Elem())}
	case reflect.Map:
		s := &Schema{Types: TypeList{"object"}}
		if t.Key().Kind() == reflect.String {
			s.AdditionalProps = inferType(t.Elem())
		}
		return s
	case reflect.Ptr:
		elem := inferType(t.Elem())
		if elem.Ref != "" {
			return &Schema{AnyOf: []*Schema{elem, {Types: TypeList{"null"}}}}
		}
		elem.Types = append(elem.Types, "null")
		return elem
	case reflect.Struct:
		if t.Name() != "" {
			return componentRef(t)
		}
		return inferStruct(t)
	default:
		// Interface fields and anything not JSON-mappable (func, chan, ...)
		// describe any value.
		return Any()
	}
}

// inferStruct builds an object schema from a struct's exported fields,
// keyed by their json tags ("-" excluded, "omitempty" making the field
// optional). Anonymous embedded structs are flattened into the parent.
func inferStruct(t reflect.Type) *Schema {
	props := map[string]*Schema{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name, opts, _ := strings.Cut(field.Tag.Get("json"), ",")

		if field.Anonymous && name == "" {
			// Anonymous embeds are flattened into the parent, matching
			// how encoding/json promotes their fields.
			embedded := field.Type
			if embedded.Kind() == reflect.Ptr {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct && embedded != timeType {
				inlined := inferStruct(embedded)
				for key, val := range inlined.Properties {
					props[key] = val
				}
				required = append(required, inlined.Required...)
			}
			continue
		}

		if field.PkgPath != "" || name == "-" {
			continue
		}

		if name == "" {
			name = field.Name
		}

		props[name] = inferType(field.Type)
		if !strings.Contains(opts, "omitempty") {
			required = append(required, name)
		}
	}

	s := &Schema{Types: TypeList{"object"}, Properties: props}
	if len(required) > 0 {
		s.Required = required
	}
	return s
}

// componentRef returns a $ref to the component registered for t, registering
// a placeholder first so recursive struct types terminate. Types whose name
// is already taken by a different type are re-qualified with the package name;
// if even that collides the struct is inlined instead of shared.
func componentRef(t reflect.Type) *Schema {
	name, fresh := componentSlot(t)
	if name == "" {
		return inferStruct(t)
	}
	if fresh {
		fillComponent(name, inferStruct(t))
	}
	return &Schema{Ref: "#/components/schemas/" + name}
}

// componentSlot reserves the component name for t and reports whether the
// caller must fill in the definition (false for already-registered types,
// which is how recursive structs terminate).
func componentSlot(t reflect.Type) (string, bool) {
	name := t.Name()

	compMu.Lock()
	defer compMu.Unlock()

	if prev, ok := compTypes[name]; ok {
		if prev == t {
			return name, false
		}

		qualified := packageName(t) + "_" + name
		if prev, ok := compTypes[qualified]; ok {
			if prev == t {
				return qualified, false
			}
			return "", false // unresolvable name clash: inline the struct
		}

		compSchemas[qualified] = &Schema{Types: TypeList{"object"}}
		compTypes[qualified] = t
		return qualified, true
	}

	compSchemas[name] = &Schema{Types: TypeList{"object"}}
	compTypes[name] = t
	return name, true
}

func fillComponent(name string, schema *Schema) {
	compMu.Lock()
	defer compMu.Unlock()
	compSchemas[name] = schema
}

func packageName(t reflect.Type) string {
	pkg := t.PkgPath()
	if i := strings.LastIndexByte(pkg, '/'); i >= 0 {
		pkg = pkg[i+1:]
	}
	return pkg
}

// registerComponent stores a hand-built component (the built-in error
// envelope); a type is not associated with it, so name clashes with inferred
// structs keep the first registration.
func registerComponent(name string, schema *Schema) {
	compMu.Lock()
	defer compMu.Unlock()
	if _, ok := compSchemas[name]; !ok {
		compSchemas[name] = schema
	}
}

// snapshotComponents copies the component registry for document assembly.
func snapshotComponents() map[string]*Schema {
	compMu.Lock()
	defer compMu.Unlock()
	out := make(map[string]*Schema, len(compSchemas))
	for name, schema := range compSchemas {
		out[name] = schema
	}
	return out
}
