package home_controller

import (
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	var currentUser *user_api.User

	var nickname string
	var signedIn bool

	apiToken, err := c.Cookie("user_api_token")
	if err == nil {
		currentUser = user_api.UserFromAPIToken(apiToken)
	}

	if currentUser == nil {
		signedIn = false
	} else {
		nickname = currentUser.Nickname
		signedIn = true
	}

	data := map[string]any{
		"Nickname": nickname,
		"SignedIn": signedIn,
	}

	resp.View(c, "home/index", data)
}
