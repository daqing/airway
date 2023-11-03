package session_page

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func SignInAction(c *gin.Context) {
	data := map[string]any{}
	resp.Page(c, "session", "sign_in", data)
}
