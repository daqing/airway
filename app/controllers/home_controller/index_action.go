package home_controller

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	resp.View(c, "home/index", nil)
}
