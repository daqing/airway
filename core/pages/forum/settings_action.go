package forum

import (
	"time"

	"github.com/daqing/airway/core/api/media_api"
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func SettingsAction(c *gin.Context) {
	rootPath := utils.PathPrefix("forum")

	token, _ := utils.CookieToken(c)
	currentUser := user_api.CurrentUser(token)
	if currentUser == nil {
		page_resp.Redirect(c, "/forum")
		return
	}

	data := map[string]any{
		"Title":     ForumTitle(),
		"Tagline":   ForumTagline(),
		"Year":      time.Now().Year(),
		"RootPath":  rootPath,
		"Session":   SessionData(currentUser),
		"AvatarURL": media_api.AssetHostPath(currentUser.Avatar),
	}

	page_resp.Page(c, "core", "forum!", "settings", data)
}
