package dashboard_page

import (
	"github.com/daqing/airway/core/pages/admin/helper"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	r.GET("", helper.CheckAdmin(IndexAction))
}
