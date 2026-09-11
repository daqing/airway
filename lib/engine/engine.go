// Package engine provides Airway's extension mechanism, inspired by Rails
// Engines. An Engine is a self-contained feature module (routes, models,
// migrations, views) shipped as an independent Go module. Host applications
// enable an Engine with a blank import; the Engine's package init registers
// it here, and the framework mounts it at boot.
package engine

import (
	"fmt"
	"io/fs"
	"sync"

	"github.com/gin-gonic/gin"
)

// Engine is a pluggable feature module. The three methods are the minimum
// contract; additional capabilities are opted into via the optional
// interfaces below.
type Engine interface {
	// Name is the unique engine identifier (e.g. "im"). Registering two
	// engines with the same name panics.
	Name() string
	// MountPath is the route prefix the engine is mounted under
	// (e.g. "/api/v1/im"). It must start with "/".
	MountPath() string
	// Routes registers the engine's routes inside its mounted group.
	Routes(r *gin.RouterGroup)
}

// Bootable engines get a Boot call after the framework infrastructure
// (database, Redis, storage) is ready, before the HTTP server starts.
type Bootable interface {
	Boot() error
}

// MigrationProvider engines ship SQL migration files (named
// <version>_<name>.up.sql / .down.sql) embedded in the binary. Install them
// into the host's db/migrate directory with `airway cli engine:install`.
type MigrationProvider interface {
	MigrationFS() fs.FS
}

// REPLModelProvider engines expose models to the interactive REPL. Keys are
// the REPL names (e.g. "Message"), values are zero-value model structs.
type REPLModelProvider interface {
	REPLModels() map[string]any
}

var (
	mu       sync.RWMutex
	registry []Engine
)

// Register adds an engine to the global registry. It is meant to be called
// from an engine package's init function. It panics on nil engines, empty
// names, invalid mount paths, or duplicate names — these are authoring
// errors that should fail fast at program start.
func Register(e Engine) {
	if e == nil {
		panic("engine: cannot register nil engine")
	}
	if e.Name() == "" {
		panic("engine: cannot register engine with empty name")
	}
	if len(e.MountPath()) == 0 || e.MountPath()[0] != '/' {
		panic(fmt.Sprintf("engine: %s mount path %q must start with /", e.Name(), e.MountPath()))
	}

	mu.Lock()
	defer mu.Unlock()

	if findLocked(e.Name()) != nil {
		panic(fmt.Sprintf("engine: duplicate engine name %q", e.Name()))
	}

	registry = append(registry, e)
}

// Engines returns the registered engines in registration order.
func Engines() []Engine {
	mu.RLock()
	defer mu.RUnlock()

	engines := make([]Engine, len(registry))
	copy(engines, registry)
	return engines
}

// Find returns the registered engine with the given name, or nil.
func Find(name string) Engine {
	mu.RLock()
	defer mu.RUnlock()

	return findLocked(name)
}

func findLocked(name string) Engine {
	for _, e := range registry {
		if e.Name() == name {
			return e
		}
	}

	return nil
}

// MountAll mounts every registered engine on the router, in registration
// order. Call it once while building the host's routes.
func MountAll(r *gin.Engine) {
	for _, e := range Engines() {
		e.Routes(r.Group(e.MountPath()))
	}
}

// BootAll calls Boot on every registered engine that implements Bootable, in
// registration order. The first error aborts the boot sequence.
func BootAll() error {
	for _, e := range Engines() {
		bootable, ok := e.(Bootable)
		if !ok {
			continue
		}

		if err := bootable.Boot(); err != nil {
			return fmt.Errorf("boot engine %s: %w", e.Name(), err)
		}
	}

	return nil
}

// REPLNamespaces merges the REPL models of every registered engine that
// implements REPLModelProvider. It returns an error when two engines (or an
// engine and one of the given reserved names) declare the same model name.
func REPLNamespaces(reserved ...string) (map[string]any, error) {
	models := map[string]any{}

	reservedSet := make(map[string]bool, len(reserved))
	for _, name := range reserved {
		reservedSet[name] = true
	}

	for _, e := range Engines() {
		provider, ok := e.(REPLModelProvider)
		if !ok {
			continue
		}

		for name, model := range provider.REPLModels() {
			if reservedSet[name] {
				return nil, fmt.Errorf("engine %s: REPL model %q conflicts with a host model", e.Name(), name)
			}
			if _, exists := models[name]; exists {
				return nil, fmt.Errorf("engine %s: duplicate REPL model %q", e.Name(), name)
			}

			models[name] = model
		}
	}

	return models, nil
}

// ResetForTest clears the registry. It is only for tests.
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()

	registry = nil
}
