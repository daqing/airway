package openapi_api

import (
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	sourceMu sync.Mutex
	source   *gin.Engine
)

// Routes registers the OpenAPI document endpoint and remembers the engine so
// the spec can enumerate the full route table on demand.
func Routes(r *gin.Engine) {
	sourceMu.Lock()
	source = r
	sourceMu.Unlock()

	r.GET("/openapi.json", SpecAction)
}
