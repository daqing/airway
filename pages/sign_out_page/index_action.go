package sign_out_page

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	resp.SetCookie(c, "user_api_token", "")

	resp.Redirect(c, "/")
}
