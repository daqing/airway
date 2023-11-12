package home_page

import (
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	var currentUser *user_api.User

	var nickname string
	var signedIn bool
	var isAdmin bool

	apiToken, err := c.Cookie("user_api_token")
	if err == nil {
		currentUser = user_api.UserFromAPIToken(apiToken)
	}

	if currentUser != nil {
		nickname = currentUser.Nickname
		signedIn = true
		isAdmin = currentUser.IsAdmin()
	}

	data := map[string]any{
		"Nickname": nickname,
		"SignedIn": signedIn,
		"IsAdmin":  isAdmin,
	}

	page_resp.Page(c, "core", "home", "index", data)
}
