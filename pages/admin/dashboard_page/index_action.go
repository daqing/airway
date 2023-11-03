package dashboard_page

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	data := map[string]any{}
	resp.Page(c, "admin/dashboard", "index", data)
}
