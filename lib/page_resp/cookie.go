package page_resp

import "github.com/gin-gonic/gin"

const weekAge = 3600 * 24 * 7

func SetCookie(c *gin.Context, name string, value string) {
	c.SetCookie(name, value, weekAge, "/", "", false, false)
}
