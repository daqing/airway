package session_page

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func DestroyAction(c *gin.Context) {
	resp.SetCookie(c, "user_api_token", "")

	resp.Redirect(c, "/")
}
