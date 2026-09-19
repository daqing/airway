package openapi

import (
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	sourceMu   sync.Mutex
	routeSetup func(*gin.Engine)
)

// RegisterRouteSource records the function that wires the full route table
// of the compiled binary onto an engine. The config package calls it from
// init(), so exactly one source exists per binary: a host project links only
// its own config, while the framework binary links the framework's. The
// openapi:generate command enumerates through this hook.
func RegisterRouteSource(setup func(*gin.Engine)) {
	sourceMu.Lock()
	defer sourceMu.Unlock()
	routeSetup = setup
}

// RouteSource returns the registered route-table setup, or nil when this
// binary has none (e.g. the standalone CLI outside a project).
func RouteSource() func(*gin.Engine) {
	sourceMu.Lock()
	defer sourceMu.Unlock()
	return routeSetup
}
