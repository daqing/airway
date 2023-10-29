package resp

import (
	"net/http"

	"github.com/daqing/airway/lib/page"
	"github.com/gin-gonic/gin"
)

func HTML(c *gin.Context, template string, data map[string]any) {
	html, err := page.Render(template, data)

	if err != nil {
		HtmlError(c, err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Data(http.StatusOK, "text/html; charset=utf-8", html)
}
