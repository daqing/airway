package models

import "reflect"

var replModels = map[string]reflect.Type{}

func registerREPLModel(name string, model any) {
	if name == "" || model == nil {
		return
	}

	replModels[name] = reflect.TypeOf(model)
}

func REPLNamespace() map[string]reflect.Type {
	namespace := make(map[string]reflect.Type, len(replModels))
	for name, modelType := range replModels {
		namespace[name] = modelType
	}

	return namespace
}
