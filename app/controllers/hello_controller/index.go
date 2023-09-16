package hello_controller

import (
	"github.com/daqing/airway/app/services"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type IndexParams struct {
	Name string `form:"name"`
}

func Index(c *gin.Context) {
	var params IndexParams

	if err := c.ShouldBind(&params); err != nil {
		utils.LogError(c, err)
	}

	hello := services.GetHello(params.Name)

	c.JSON(200, gin.H{"ok": true, "hello": hello})
}
