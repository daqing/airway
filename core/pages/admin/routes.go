package admin

import (
	"github.com/daqing/airway/core/pages/admin/admin_menu"
	"github.com/daqing/airway/core/pages/admin/admin_node"
	"github.com/daqing/airway/core/pages/admin/dashboard_page"
	"github.com/daqing/airway/core/pages/admin/post_page"
	"github.com/daqing/airway/core/pages/admin/user_page"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	g := r.Group("/admin")
	{
		dashboard_page.Routes(g)

		user_page.Routes(g)
		post_page.Routes(g)
		admin_node.Routes(g)
		admin_menu.Routes(g)
	}
}
