package home_page

import "github.com/gin-gonic/gin"

func Routes(r *gin.Engine) {
	r.GET("/", IndexAction)
}
