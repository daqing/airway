package node_page

import (
	"github.com/daqing/airway/core/pages/admin/helper"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/node")
	{
		g.GET("", helper.CheckAdmin(IndexAction))
		g.GET("/new", helper.CheckAdmin(NewAction))
		g.GET("/edit", helper.CheckAdmin(EditAction))

		g.POST("/create", helper.CheckAdmin(CreateAction))
		g.POST("/update", helper.CheckAdmin(UpdateAction))

		// TODO: csrf protection
		g.GET("/delete", helper.CheckAdmin(DeleteAction))
	}
}
