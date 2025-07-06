package resp

import (
	"github.com/gin-gonic/gin"
)

func Empty(c *gin.Context) {
	OK(c, nil)
}

func OK(c *gin.Context, data any) {
	c.JSON(200, gin.H{"code": 0, "data": data, "message": ""})
}
