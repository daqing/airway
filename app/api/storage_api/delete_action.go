package storage_api

import (
	"os"

	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

type DeleteParams struct {
	Path string `json:"path"`
}

func DeleteAction(c *gin.Context) {
	var p DeleteParams

	if err := c.ShouldBindJSON(&p); err != nil {
		resp.Error(c, err)
		return
	}

	fullPath := fullPath(p.Path)
	// if file not exists, return ok
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		resp.OK(c, p.Path)
		return
	}

	if err := os.Remove(fullPath); err != nil {
		resp.Error(c, err)
		return
	}

	resp.OK(c, p.Path)
}
