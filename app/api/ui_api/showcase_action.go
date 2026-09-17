package ui_api

import (
	"github.com/gin-gonic/gin"

	"github.com/daqing/airway/app/views/ui"
	"github.com/daqing/airway/lib/render"
)

// ShowcaseAction serves the airway-ui component showcase at /ui.
func ShowcaseAction(c *gin.Context) {
	render.HTML(c, ui.Showcase())
}

// DemoItemsAction backs the showcase's data-layer demo: a JSON endpoint
// using the standard render envelope ({"code":0,"data":…}).
func DemoItemsAction(c *gin.Context) {
	render.OK(c, gin.H{
		"items": []gin.H{
			{"id": 1, "name": "SSD", "stock": 42},
			{"id": 2, "name": "Keyboard", "stock": 7},
			{"id": 3, "name": "Monitor", "stock": 15},
		},
	})
}
