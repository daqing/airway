package session_page

import (
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/api_resp"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type CreateSessionParams struct {
	Username string `form:"username"`
	Password string `form:"password"`
}

func CreateAction(c *gin.Context) {
	var p CreateSessionParams

	if err := c.ShouldBind(&p); err != nil {
		api_resp.Error(c, err)
		return
	}

	username := utils.TrimFull(p.Username)
	password := utils.TrimFull(p.Password)

	if len(username) == 0 || len(password) == 0 {
		// empty request
		page_resp.Redirect(c, "/session/sign_in")
		return
	}

	user, err := user_api.LoginUser(
		[]repo.KVPair{repo.KV("username", username)},
		password,
	)

	if err != nil || user == nil {
		page_resp.Redirect(c, "/session/sign_in")
		return
	}

	// login ok, set cookie
	page_resp.SetCookie(c, "user_api_token", user.APIToken)
	page_resp.Redirect(c, "/")
}
