package forum

import (
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/utils"
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

func SessionData(token string) map[string]any {
	var currentUser *user_api.User
	var nickname string
	var signedIn bool
	var isAdmin bool

	currentUser = user_api.UserFromAPIToken(token)

	if currentUser != nil {
		nickname = currentUser.Nickname
		signedIn = true
		isAdmin = currentUser.IsAdmin()
	}

	return map[string]any{
		"CurrentUser": currentUser,
		"Nickname":    nickname,
		"SignedIn":    signedIn,
		"IsAdmin":     isAdmin,
	}
}
