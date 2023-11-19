package admin

import (
	"github.com/daqing/airway/core/pages/admin/admin_dashboard"
	"github.com/daqing/airway/core/pages/admin/admin_menu"
	"github.com/daqing/airway/core/pages/admin/admin_node"
	"github.com/daqing/airway/core/pages/admin/admin_post"
	"github.com/daqing/airway/core/pages/admin/admin_user"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	g := r.Group("/admin")
	{
		admin_dashboard.Routes(g)

		admin_user.Routes(g)
		admin_post.Routes(g)
		admin_node.Routes(g)
		admin_menu.Routes(g)
	}
}
