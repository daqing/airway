package resp

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func ErrorMsg(c *gin.Context, msg string) {
	fmt.Printf("===> Response Error: %s", msg)

	c.JSON(200, gin.H{"ok": false, "data": gin.H{}, "message": msg})
	c.Abort()
}

func Error(c *gin.Context, err error) {
	ErrorMsg(c, err.Error())
}

func HtmlError(c *gin.Context, err error) {
	c.String(500, fmt.Sprintf("ERR: %s", err.Error()))
}
