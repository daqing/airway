package up_api

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func UpAction(c *gin.Context) {
	fmt.Fprintf(c.Writer, "UP\n")
}
