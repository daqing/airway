package asset_api

import (
	"errors"
	"path/filepath"

	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func ServeAction(c *gin.Context) {
	path := c.Param("path")
	if filepath.Ext(path) == "" {
		resp.Error(c, errors.New("404 Not Found"))
		return
	}

	storageRoot := utils.GetEnvMust("STORAGE_ROOT")
	fullPath := filepath.Join(storageRoot, path)

	c.File(fullPath)
}
