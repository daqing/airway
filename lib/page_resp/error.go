package page_resp

import "github.com/gin-gonic/gin"

func Error(c *gin.Context, err error) {
	c.AbortWithError(500, err)
}
