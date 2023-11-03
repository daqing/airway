package session_page

import (
	"github.com/daqing/airway/lib/page_resp"
	"github.com/gin-gonic/gin"
)

func SignInAction(c *gin.Context) {
	data := map[string]any{}
	page_resp.Page(c, "session", "sign_in", data)
}
