package page_resp

import (
	"fmt"
	"strings"

	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

const DOT = "."

// Plain renders page without adding "_page" suffix
func Expand(c *gin.Context, topDir, page, action string, obj map[string]any) {
	pwd := utils.GetEnvMust("AIRWAY_PWD")

	var pageDir = page

	// expand "forum.home" to "forum/forum_home"
	if strings.Contains(page, DOT) {
		parts := strings.Split(page, DOT)
		pageDir = fmt.Sprintf("%s/%s_%s", parts[0], parts[0], parts[1])
	}

	prefix := fmt.Sprintf("%s/%s/pages/%s", pwd, topDir, pageDir)

	renderTemplate(c, prefix, action, obj)
}
