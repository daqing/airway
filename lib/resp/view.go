package resp

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func View(c *gin.Context, view string, data map[string]any) {
	pwd := os.Getenv("AIRWAY_PWD")
	prefix := fmt.Sprintf("%s/app/views", pwd)

	renderTemplate(c, prefix, view, data)
}
