package plugin

import (
	"fmt"
	"sort"
	"sync"
)

var registry = struct {
	sync.RWMutex
	plugins map[string]Plugin
}{plugins: map[string]Plugin{}}

// Register adds a statically linked plugin after validating its public manifest.
// Plugins normally call this from their package init function.
func Register(candidate Plugin) error {
	if candidate == nil {
		return fmt.Errorf("plugin is nil")
	}
	manifest := candidate.Manifest()
	if err := manifest.Validate(CoreVersion); err != nil {
		return err
	}
	registry.Lock()
	defer registry.Unlock()
	if _, exists := registry.plugins[manifest.Name]; exists {
		return fmt.Errorf("plugin %q is already registered", manifest.Name)
	}
	registry.plugins[manifest.Name] = candidate
	return nil
}

// Registered returns a stable snapshot ordered by plugin name.
func Registered() []Plugin {
	registry.RLock()
	defer registry.RUnlock()
	names := make([]string, 0, len(registry.plugins))
	for name := range registry.plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]Plugin, 0, len(names))
	for _, name := range names {
		result = append(result, registry.plugins[name])
	}
	return result
}
