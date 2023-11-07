package page_resp

import "github.com/gin-gonic/gin"

func Error(c *gin.Context, err error) {
	ErrorMsg(c, err.Error())
}

func ErrorMsg(c *gin.Context, msg string) {
	c.String(200, "Err: %s", msg)
}
