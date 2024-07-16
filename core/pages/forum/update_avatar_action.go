package forum

import (
	"fmt"

	"github.com/daqing/airway/app/models"
	"github.com/daqing/airway/core/api/media_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/sql_orm"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func UpdateAvatarAction(c *gin.Context) {
	token, err := utils.CookieToken(c)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	currentUser := user_api.CurrentUser(token)
	if currentUser == nil {
		page_resp.Redirect(c, "/")
		return
	}

	fileHeader, _ := c.FormFile("avatar")

	destPath, filePath, err := media_api.DestFilePath(fileHeader)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	// TODO: add image processing

	c.SaveUploadedFile(fileHeader, destPath)

	ok := sql_orm.UpdateFields[models.User](
		currentUser.ID,
		[]sql_orm.KVPair{
			sql_orm.KV("avatar", filePath),
		},
	)

	if !ok {
		page_resp.Error(c, fmt.Errorf("update user avatar field failed"))
		return
	}

	page_resp.Redirect(c, "/forum/settings")
}
