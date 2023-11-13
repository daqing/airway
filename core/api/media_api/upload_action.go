package media_api

import (
	"fmt"
	"os"

	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/api_resp"

	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func UploadAction(c *gin.Context) {
	currentUser := user_api.CurrentUser(utils.AuthToken(c))

	// user must be logged in to upload files
	if currentUser == nil {
		api_resp.Error(c, fmt.Errorf("current user not found"))
		return
	}

	file, _ := c.FormFile("file")

	f, err := file.Open()
	if err != nil {
		api_resp.Error(c, err)
		return
	}

	hash, err := utils.MD5SumFile(f)
	if err != nil {
		api_resp.Error(c, err)
		return
	}

	newFilename := replace(file.Filename, hash)
	mime := file.Header.Get("Content-Type")

	row, err := SaveFile(currentUser.Id, newFilename, mime, file.Size)
	if err != nil {
		api_resp.Error(c, err)
		return
	}

	if row == nil {
		// failed to create database row
		api_resp.Error(c, fmt.Errorf("failed to create database row"))
		return
	}

	// move uplaoded file to asset directory
	assetDir, err := utils.GetEnv("AIRWAY_STORAGE_DIR")
	if err != nil {
		api_resp.Error(c, err)
		return
	}

	destDir := hashDirPath(assetDir, newFilename)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		api_resp.Error(c, err)
		return
	}

	destFile := destDir + "/" + newFilename

	c.SaveUploadedFile(file, destFile)

	assetHost, err := utils.GetEnv("AIRWAY_ASSET_HOST")
	if err != nil {
		api_resp.Error(c, err)
		return
	}

	api_resp.OK(c, gin.H{"filename": newFilename, "url": assetHostPath(assetHost, newFilename)})
}
