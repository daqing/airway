package dashboard_page

import (
	"github.com/daqing/airway/lib/page_resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	data := map[string]any{}
	page_resp.Page(c, "admin/dashboard", "index", data)
}
