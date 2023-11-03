package helper

import (
	"github.com/daqing/airway/api/user_api"
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func CheckAdmin(action gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("user_api_token")
		if err != nil {
			// c.AbortWithError(500, err)
			resp.Redirect(c, "/")
			return
		}

		var admin = user_api.CurrentAdmin(token)
		if admin == nil {
			resp.Redirect(c, "/")
			return
		}

		c.Set("admin", admin)

		action(c)
	}
}
