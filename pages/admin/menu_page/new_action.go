package menu_page

import (
	"github.com/daqing/airway/lib/page_resp"
	"github.com/gin-gonic/gin"
)

func NewAction(c *gin.Context) {
	page_resp.Page(c, "admin/menu", "new", nil)
}
