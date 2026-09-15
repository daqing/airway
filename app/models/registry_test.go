package models

import (
	"reflect"
	"testing"
)

type registryTestModel struct {
	ID int64 `db:"id"`
}

func (registryTestModel) TableName() string {
	return "registry_test_models"
}

func TestRegisterREPLModelAddsToNamespace(t *testing.T) {
	registerREPLModel("RegistryTestModel", registryTestModel{})

	modelType, ok := REPLNamespace()["RegistryTestModel"]
	if !ok {
		t.Fatalf("expected RegistryTestModel in REPL namespace, got %#v", REPLNamespace())
	}

	if modelType != reflect.TypeOf(registryTestModel{}) {
		t.Fatalf("expected type %v, got %v", reflect.TypeOf(registryTestModel{}), modelType)
	}
}
