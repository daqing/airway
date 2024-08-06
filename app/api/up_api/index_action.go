package up_api

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	fmt.Fprintf(c.Writer, "UP\n")
}
