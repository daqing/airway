package page_resp

import "github.com/gin-gonic/gin"

func Redirect(c *gin.Context, path string) {
	c.Redirect(302, path)
}
