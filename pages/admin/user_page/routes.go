package user_page

import (
	"github.com/daqing/airway/pages/admin/helper"
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.RouterGroup) {
	g := r.Group("/user")
	{
		g.GET("", helper.CheckAdmin(IndexAction))
		g.GET("/new", helper.CheckAdmin(NewAction))
		g.POST("/create", helper.CheckAdmin(CreateAction))
	}
}