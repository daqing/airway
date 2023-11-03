package sign_in_page

import (
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type AuthParams struct {
	Username string `form:"username"`
	Password string `form:"password"`
}

func AuthAction(c *gin.Context) {
	var p AuthParams

	if err := c.ShouldBind(&p); err != nil {
		resp.Error(c, err)
		return
	}

	username := utils.TrimFull(p.Username)
	password := utils.TrimFull(p.Password)

	if len(username) == 0 || len(password) == 0 {
		// empty request
		resp.Redirect(c, "/sign_in/index")
		return
	}

	user, err := user_api.LoginUser(username, password)
	if err != nil || user == nil {
		resp.Redirect(c, "/sign_in/index")
		return
	}

	// login ok, set cookie
	resp.SetCookie(c, "user_api_token", user.ApiToken)
	resp.Redirect(c, "/")
}
