package dashboard_page

import (
	"github.com/daqing/airway/pages/admin/helper"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/dashboard")
	{
		g.GET("", helper.CheckAdmin(IndexAction))
	}
}
