package openapi

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// resetRegistry isolates the package-level declaration state between tests.
func resetRegistry() {
	declMu.Lock()
	declared = map[string]*Operation{}
	docDecl = nil
	declMu.Unlock()

	compMu.Lock()
	compSchemas = map[string]*Schema{}
	compTypes = map[string]reflect.Type{}
	compMu.Unlock()
}

func marshalCompact(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(raw)
}

type openapiTestPost struct {
	ID        int64          `json:"id"`
	Title     string         `json:"title"`
	Excerpt   string         `json:"excerpt,omitempty"`
	Views     *int           `json:"views"`
	Tags      []string       `json:"tags"`
	Meta      map[string]any `json:"meta,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

func TestInferStructSchema(t *testing.T) {
	resetRegistry()

	schema := Item[openapiTestPost]()
	if schema.Ref != "#/components/schemas/openapiTestPost" {
		t.Fatalf("expected $ref, got %q", schema.Ref)
	}

	expected := `{"type":"object","properties":{` +
		`"created_at":{"type":"string","format":"date-time"},` +
		`"excerpt":{"type":"string"},` +
		`"id":{"type":"integer","format":"int64"},` +
		`"meta":{"type":"object","additionalProperties":{}},` +
		`"tags":{"type":"array","items":{"type":"string"}},` +
		`"title":{"type":"string"},` +
		`"views":{"type":["integer","null"]}},` +
		`"required":["id","title","views","tags","created_at"]}`

	got := marshalCompact(t, snapshotComponents()["openapiTestPost"])
	if got != expected {
		t.Fatalf("expected schema %s, got %s", expected, got)
	}
}

func TestTypeListMarshaling(t *testing.T) {
	if got := marshalCompact(t, TypeList{"string"}); got != `"string"` {
		t.Fatalf("single type should marshal as string, got %s", got)
	}
	if got := marshalCompact(t, TypeList{"string", "null"}); got != `["string","null"]` {
		t.Fatalf("nullable type should marshal as array, got %s", got)
	}
}

func TestComponentNameCollision(t *testing.T) {
	resetRegistry()

	type colliding struct {
		Name string `json:"name"`
	}

	// Simulate a foreign type claiming the plain name first.
	compMu.Lock()
	compTypes["colliding"] = reflect.TypeOf(time.Time{})
	compMu.Unlock()

	schema := Item[colliding]()
	if schema.Ref != "#/components/schemas/openapi_colliding" {
		t.Fatalf("expected package-qualified ref, got %q", schema.Ref)
	}
}

func TestRecursiveStructTerminates(t *testing.T) {
	resetRegistry()

	type node struct {
		Name     string  `json:"name"`
		Children []*node `json:"children,omitempty"`
	}

	schema := Item[node]()
	if schema.Ref == "" {
		t.Fatalf("expected $ref for named struct")
	}

	component := snapshotComponents()["node"]
	got := marshalCompact(t, component)
	expected := `{"type":"object","properties":{` +
		`"children":{"type":"array","items":{"anyOf":[{"$ref":"#/components/schemas/node"},{"type":"null"}]}},` +
		`"name":{"type":"string"}},` +
		`"required":["name"]}`
	if got != expected {
		t.Fatalf("expected recursive schema %s, got %s", expected, got)
	}
}

func TestObjHelper(t *testing.T) {
	got := marshalCompact(t, Obj(map[string]*Schema{"error": Str()}))
	expected := `{"type":"object","properties":{"error":{"type":"string"}}}`
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

func TestNullableScalarUsesTypeArray(t *testing.T) {
	var count *int64
	got := marshalCompact(t, inferType(reflect.TypeOf(count)))
	expected := `{"type":["integer","null"],"format":"int64"}`
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}
