package home_page

import "github.com/gin-gonic/gin"

func IndexAction(c *gin.Context) {
	c.HTML(200, "home/index", gin.H{
		"title": "Welcome to Airway",
	})
}
