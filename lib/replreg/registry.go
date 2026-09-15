// Package replreg holds the process-wide registry of models exposed to the
// interactive REPL. Host apps and plugins register their models here (usually
// from init functions); the REPL builds its namespace from the registry.
package replreg

import (
	"reflect"
	"sync"
)

var (
	mu     sync.RWMutex
	models = map[string]reflect.Type{}
)

// Register adds a model to the REPL namespace under name. Empty names and nil
// models are ignored.
func Register(name string, model any) {
	if name == "" || model == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	models[name] = reflect.TypeOf(model)
}

// Namespace returns a copy of the registered name -> model type mapping.
func Namespace() map[string]reflect.Type {
	mu.RLock()
	defer mu.RUnlock()

	namespace := make(map[string]reflect.Type, len(models))
	for name, modelType := range models {
		namespace[name] = modelType
	}

	return namespace
}
