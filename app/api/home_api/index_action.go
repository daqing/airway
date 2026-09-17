package home_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/views/home"
	"github.com/daqing/airway/lib/render"
)

func IndexAction(c *gin.Context) {
	// counterStart feeds the demo island's initial props — the island
	// pattern: server renders the skeleton, props ride along as JSON.
	render.HTML(c, home.Index(3))
}
