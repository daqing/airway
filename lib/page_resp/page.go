package page_resp

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

func Page(c *gin.Context, topDir, page string, action string, obj map[string]any) {
	pwd := os.Getenv("AIRWAY_PWD")
	prefix := fmt.Sprintf("%s/%s/pages/%s_page", pwd, topDir, page)

	renderTemplate(c, prefix, action, obj)
}
