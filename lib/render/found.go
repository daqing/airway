package render

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 302 Found response
func Found(c *gin.Context, url string) {
	c.Redirect(http.StatusFound, url)
}
