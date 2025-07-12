package storage_api

import (
	"os"
	"path/filepath"

	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func UploadAction(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		resp.Error(c, err)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		resp.Error(c, err)
		return
	}

	md5Hash, err := utils.MD5SumFile(file)
	if err != nil {
		resp.Error(c, err)
		return
	}

	extName := filepath.Ext(fileHeader.Filename)
	filePath := utils.FilePathFromMD5(md5Hash, extName)

	fullPath := fullPath(filePath)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		resp.Error(c, err)
		return
	}

	c.SaveUploadedFile(fileHeader, fullPath)

	resp.OK(c, filePath)
}

func fullPath(filePath string) string {
	storageRoot := utils.GetEnvMust("STORAGE_ROOT")

	return filepath.Join(storageRoot, filePath)
}
