// Package plugin provides Airway's extension mechanism, inspired by
// WordPress plugins. A Plugin is a self-contained feature module (routes,
// models, migrations, views) shipped as an independent Go module. Host
// applications enable a Plugin with a blank import; the Plugin's package
// init registers it here, and the framework mounts it at boot.
package plugin

import (
	"fmt"
	"io/fs"
	"sync"

	"github.com/gin-gonic/gin"
)

// Plugin is a pluggable feature module. The three methods are the minimum
// contract; additional capabilities are opted into via the optional
// interfaces below.
type Plugin interface {
	// Name is the unique plugin identifier (e.g. "im"). Registering two
	// plugins with the same name panics.
	Name() string
	// MountPath is the route prefix the plugin is mounted under
	// (e.g. "/api/v1/im"). It must start with "/".
	MountPath() string
	// Routes registers the plugin's routes inside its mounted group.
	Routes(r *gin.RouterGroup)
}

// Bootable plugins get a Boot call after the framework infrastructure
// (database, Redis, storage) is ready, before the HTTP server starts.
type Bootable interface {
	Boot() error
}

// MigrationProvider plugins ship SQL migration files (named
// <version>_<name>.up.sql / .down.sql) embedded in the binary. Install them
// into the host's db/migrate directory with `airway plugin:install`.
type MigrationProvider interface {
	MigrationFS() fs.FS
}

// REPLModelProvider plugins expose models to the interactive REPL. Keys are
// the REPL names (e.g. "Message"), values are zero-value model structs.
type REPLModelProvider interface {
	REPLModels() map[string]any
}

var (
	mu       sync.RWMutex
	registry []Plugin
)

// Register adds a plugin to the global registry. It is meant to be called
// from a plugin package's init function. It panics on nil plugins, empty
// names, invalid mount paths, or duplicate names — these are authoring
// errors that should fail fast at program start.
func Register(p Plugin) {
	if p == nil {
		panic("plugin: cannot register nil plugin")
	}
	if p.Name() == "" {
		panic("plugin: cannot register plugin with empty name")
	}
	if len(p.MountPath()) == 0 || p.MountPath()[0] != '/' {
		panic(fmt.Sprintf("plugin: %s mount path %q must start with /", p.Name(), p.MountPath()))
	}

	mu.Lock()
	defer mu.Unlock()

	if findLocked(p.Name()) != nil {
		panic(fmt.Sprintf("plugin: duplicate plugin name %q", p.Name()))
	}

	registry = append(registry, p)
}

// Plugins returns the registered plugins in registration order.
func Plugins() []Plugin {
	mu.RLock()
	defer mu.RUnlock()

	plugins := make([]Plugin, len(registry))
	copy(plugins, registry)
	return plugins
}

// Find returns the registered plugin with the given name, or nil.
func Find(name string) Plugin {
	mu.RLock()
	defer mu.RUnlock()

	return findLocked(name)
}

func findLocked(name string) Plugin {
	for _, p := range registry {
		if p.Name() == name {
			return p
		}
	}

	return nil
}

// MountAll mounts every registered plugin on the router, in registration
// order. Call it once while building the host's routes.
func MountAll(r *gin.Engine) {
	for _, p := range Plugins() {
		p.Routes(r.Group(p.MountPath()))
	}
}

// BootAll calls Boot on every registered plugin that implements Bootable, in
// registration order. The first error aborts the boot sequence.
func BootAll() error {
	for _, p := range Plugins() {
		bootable, ok := p.(Bootable)
		if !ok {
			continue
		}

		if err := bootable.Boot(); err != nil {
			return fmt.Errorf("boot plugin %s: %w", p.Name(), err)
		}
	}

	return nil
}

// REPLNamespaces merges the REPL models of every registered plugin that
// implements REPLModelProvider. It returns an error when two plugins (or a
// plugin and one of the given reserved names) declare the same model name.
func REPLNamespaces(reserved ...string) (map[string]any, error) {
	models := map[string]any{}

	reservedSet := make(map[string]bool, len(reserved))
	for _, name := range reserved {
		reservedSet[name] = true
	}

	for _, p := range Plugins() {
		provider, ok := p.(REPLModelProvider)
		if !ok {
			continue
		}

		for name, model := range provider.REPLModels() {
			if reservedSet[name] {
				return nil, fmt.Errorf("plugin %s: REPL model %q conflicts with a host model", p.Name(), name)
			}
			if _, exists := models[name]; exists {
				return nil, fmt.Errorf("plugin %s: duplicate REPL model %q", p.Name(), name)
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
