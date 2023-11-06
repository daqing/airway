package admin

import (
	"github.com/daqing/airway/pages/admin/dashboard_page"
	"github.com/daqing/airway/pages/admin/node_page"
	"github.com/daqing/airway/pages/admin/post_page"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	g := r.Group("/admin")
	{
		dashboard_page.Routes(g)

		post_page.Routes(g)
		node_page.Routes(g)
	}
}
