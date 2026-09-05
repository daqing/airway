package home_api

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/views/home"
	"github.com/daqing/airway/lib/render"
)

func IndexAction(c *gin.Context) {
	render.HTML(c,
		home.Index("Airway works!", time.Now().String()),
	)
}
