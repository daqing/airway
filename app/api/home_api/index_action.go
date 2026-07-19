package home_api

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func IndexAction(c *gin.Context) {
	fmt.Fprintf(c.Writer, "Hello, Airway!")
}
