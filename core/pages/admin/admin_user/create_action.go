package admin_user

import (
	"fmt"

	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type CreateParams struct {
	Nickname string `form:"nickname"`
	Username string `form:"username"`
	Password string `form:"password"`
}

func CreateAction(c *gin.Context) {
	var p CreateParams

	if err := c.ShouldBind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	nickname := utils.TrimFull(p.Nickname)
	username := utils.TrimFull(p.Username)
	password := utils.TrimFull(p.Password)

	if len(nickname) == 0 || len(username) == 0 {
		page_resp.Error(c, fmt.Errorf("title or content must not be empty"))
		return
	}

	if len(password) == 0 {
		page_resp.Error(c, fmt.Errorf("password must not be empty"))
		return
	}

	_, err := user_api.CreateBasicUser(nickname, username, password)
	if err != nil {
		page_resp.Error(c, err)
		return
	}

	page_resp.Redirect(c, "/admin/user")
}
