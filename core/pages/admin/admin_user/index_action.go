package admin_user

import (
	"github.com/daqing/airway/core/api/user_api"
	"github.com/daqing/airway/lib/page_resp"
	"github.com/daqing/airway/models"
	"github.com/gin-gonic/gin"
)

type UserItem struct {
	ID       uint
	Nickname string
	Username string
	RoleName string
	APIToken string
	Balance  string
}

type IndexParams struct {
	Page int `form:"page"`
}

func IndexAction(c *gin.Context) {
	var p IndexParams

	if err := c.Bind(&p); err != nil {
		page_resp.Error(c, err)
		return
	}

	users, err := user_api.Users(
		[]string{"id", "nickname", "username", "role", "api_token", "balance"},
		"id DESC",
		p.Page,
		50,
	)

	if err != nil {
		page_resp.Error(c, err)
		return
	}

	var items []*UserItem

	for _, user := range users {
		items = append(items,
			&UserItem{
				ID:       user.ID,
				Nickname: user.Nickname,
				Username: user.Username,
				RoleName: models.RoleName(user.Role),
				APIToken: user.APIToken,
				Balance:  user.Balance.Yuan(),
			})
	}

	data := map[string]any{
		"Users": items,
	}

	page_resp.Page(c, "core", "admin.user", "index", data)
}
