package resp

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func Page(c *gin.Context, page string, action string, obj map[string]any) {
	pwd := os.Getenv("AIRWAY_PWD")
	prefix := fmt.Sprintf("%s/pages/%s_page", pwd, page)

	renderTemplate(c, prefix, action, obj)
}
