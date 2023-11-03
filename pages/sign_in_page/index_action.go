package sign_in_page

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	resp.Page(c, "sign_in", "index", nil)
}
