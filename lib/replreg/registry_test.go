package replreg

import (
	"reflect"
	"testing"
)

func TestRegisterAndNamespace(t *testing.T) {
	type widget struct {
		ID int64 `db:"id"`
	}

	Register("Widget", widget{})
	Register("", widget{})
	Register("Nil", nil)

	namespace := Namespace()

	modelType, ok := namespace["Widget"]
	if !ok {
		t.Fatalf("expected Widget in namespace, got %#v", namespace)
	}

	if modelType != reflect.TypeOf(widget{}) {
		t.Fatalf("expected Widget type %v, got %v", reflect.TypeOf(widget{}), modelType)
	}

	if _, ok := namespace["Nil"]; ok {
		t.Fatalf("nil model must not be registered, got %#v", namespace)
	}

	namespace["Widget"] = reflect.TypeOf(0)
	if Namespace()["Widget"] != reflect.TypeOf(widget{}) {
		t.Fatalf("Namespace must return a copy")
	}
}
