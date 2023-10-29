package resp

import (
	"maps"
	"net/http"
	"time"

	"github.com/daqing/airway/lib/page"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

func HTML(c *gin.Context, template string, obj map[string]any) {
	var data = defaultData()

	if obj == nil {
		obj = data
	} else {
		maps.Copy(obj, data)
	}

	html, err := page.Render(template, obj)

	if err != nil {
		HtmlError(c, err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Data(http.StatusOK, "text/html; charset=utf-8", html)
}

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
