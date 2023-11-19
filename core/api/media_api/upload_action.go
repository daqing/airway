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

	fileHeader, _ := c.FormFile("file")

	file, err := fileHeader.Open()
	if err != nil {
		api_resp.Error(c, err)
		return
	}

	hash, err := utils.MD5SumFile(file)
	if err != nil {
		api_resp.Error(c, err)
		return
	}

	newFilename := replace(fileHeader.Filename, hash)
	mime := fileHeader.Header.Get("Content-Type")

	row, err := SaveFile(currentUser.Id, newFilename, mime, fileHeader.Size)
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
	destDir := hashDirPath(AssetStorageDir(), newFilename)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		api_resp.Error(c, err)
		return
	}

	destFile := destDir + "/" + newFilename

	c.SaveUploadedFile(fileHeader, destFile)

	api_resp.OK(c, gin.H{"filename": newFilename, "url": AssetHostPath(newFilename)})
}
