package session_page

import (
	"github.com/daqing/airway/lib/page_resp"
	"github.com/gin-gonic/gin"
)

func DestroyAction(c *gin.Context) {
	page_resp.SetCookie(c, "user_api_token", "")

	page_resp.Redirect(c, "/")
}
