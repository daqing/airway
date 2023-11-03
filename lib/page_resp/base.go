package page_resp

import (
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/daqing/airway/lib/tmpl"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func defaultData() map[string]any {
	var data = make(map[string]any)

	var config = utils.AppConfig()

	if config.IsLocal {
		data["ts"] = time.Now().UnixNano()
	} else {
		data["ts"] = 1
	}

	return data
}

func renderTemplate(c *gin.Context, prefix string, template string, obj map[string]any) {
	var data = defaultData()

	if obj == nil {
		obj = data
	} else {
		maps.Copy(obj, data)
	}

	html, err := tmpl.Render(prefix, template, obj)

	if err != nil {
		HtmlError(c, err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Data(http.StatusOK, "text/html; charset=utf-8", html)
}

func HtmlError(c *gin.Context, err error) {
	c.String(500, fmt.Sprintf("ERR: %s", err.Error()))
}
