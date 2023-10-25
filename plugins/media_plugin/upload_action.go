package media_plugin

import (
	"fmt"
	"os"

	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/plugins/user_plugin"
	"github.com/gin-gonic/gin"
)

func UploadAction(c *gin.Context) {
	currentUser := user_plugin.CurrentUser(utils.AuthToken(c))

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
	}

	// move uplaoded file to asset directory
	destDir := hashDirPath(newFilename)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		resp.Error(c, err)
		return
	}

	destFile := destDir + "/" + newFilename

	c.SaveUploadedFile(file, destFile)

	resp.OK(c, gin.H{"filename": newFilename, "url": assetHostPath(newFilename)})
}
