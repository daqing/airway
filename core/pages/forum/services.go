package forum

import (
	"github.com/daqing/airway/core/api/media_api"
	"github.com/daqing/airway/lib/utils"
	"github.com/daqing/airway/models"
)

func ForumTitle() string {
	return utils.GetEnvMust("AW_FORUM_TITLE")
}

func ForumTagline() string {
	tagline, err := utils.GetEnv("AW_FORUM_TAGLINE")
	if err != nil {
		return ""
	}

	return tagline
}

func SessionData(currentUser *models.User) map[string]any {
	var nickname string
	var signedIn bool
	var isAdmin bool
	var avatarURL string

	if currentUser != nil {
		nickname = currentUser.Nickname
		signedIn = true
		isAdmin = currentUser.IsAdmin()
		avatarURL = media_api.AssetHostPath(currentUser.Avatar)
	}

	return map[string]any{
		"CurrentUser": currentUser,
		"Nickname":    nickname,
		"SignedIn":    signedIn,
		"IsAdmin":     isAdmin,
		"AvatarURL":   avatarURL,
	}
}
