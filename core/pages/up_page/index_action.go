package up_page

import (
	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	c.String(200, "UP")
}
