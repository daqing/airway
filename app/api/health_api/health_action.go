package health_api

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func HealthAction(c *gin.Context) {
	fmt.Fprintf(c.Writer, "UP\n")
}
