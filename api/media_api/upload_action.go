package media_api

import (
	"fmt"
	"os"

	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func UploadAction(c *gin.Context) {
	currentUser := user_api.CurrentUser(utils.AuthToken(c))

	// user must be logged in to upload files
	if currentUser == nil {
		resp.Error(c, fmt.Errorf("current user not found"))
		return
	}

	file, _ := c.FormFile("file")

	f, err := file.Open()
	if err != nil {
		resp.Error(c, err)
		return
	}

	hash, err := utils.MD5SumFile(f)
	if err != nil {
		resp.Error(c, err)
		return
	}

	newFilename := replace(file.Filename, hash)
	mime := file.Header.Get("Content-Type")

	row, err := SaveFile(currentUser.Id, newFilename, mime, file.Size)
	if err != nil {
		resp.Error(c, err)
		return
	}

	if row == nil {
		// failed to create database row
		resp.Error(c, fmt.Errorf("failed to create database row"))
		return
	}

	// move uplaoded file to asset directory
	assetDir := os.Getenv("AIRWAY_STORAGE_DIR")
	if assetDir == "" {
		resp.Error(c, fmt.Errorf(
			"no environment variable defined for AIRWAY_STORAGE_DIR",
		))
		return
	}

	destDir := hashDirPath(assetDir, newFilename)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		resp.Error(c, err)
		return
	}

	destFile := destDir + "/" + newFilename

	c.SaveUploadedFile(file, destFile)

	assetHost := os.Getenv("AIRWAY_ASSET_HOST")
	if assetHost == "" {
		resp.Error(c, fmt.Errorf(
			"no environment variable defined for AIRWAY_ASSET_HOST",
		))

		return
	}

	resp.OK(c, gin.H{"filename": newFilename, "url": assetHostPath(assetHost, newFilename)})
}
