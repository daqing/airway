package openapi_api

import (
	"net/http"
	"os"
	"strings"

	"github.com/daqing/airway/lib/openapi"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

// SpecAction serves the OpenAPI 3.2 document for the running server; the
// servers entry is derived from the request so generated clients point at
// the same host and deployment prefix.
func SpecAction(c *gin.Context) {
	sourceMu.Lock()
	engine := source
	sourceMu.Unlock()

	if engine == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "openapi route table is not ready"})
		return
	}

	body, err := openapi.Build(engine, openapi.BuildOptions{
		Version: docVersion(),
		Server:  requestServer(c),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/json", body)
}

func requestServer(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.Request.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}

	return scheme + "://" + c.Request.Host + utils.URLPrefix()
}

// docVersion reads the project's VERSION file; the binary embeds it under
// package main, which this package cannot import.
func docVersion() string {
	if raw, err := os.ReadFile("VERSION"); err == nil {
		return strings.TrimSpace(string(raw))
	}

	return "dev"
}
