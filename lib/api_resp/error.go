package api_resp

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

const DefaultMessage = "===> Got server error: "

func LogError(c *gin.Context, err error) {
	LogErrorMsg(c, err, DefaultMessage)
}

func LogErrorMsg(c *gin.Context, err error, message string) {
	fmt.Println(message, err.Error())

	Error(c, err)

	panic(message + err.Error())
}

func LogInvalidUser(c *gin.Context) {
	LogError(c, fmt.Errorf("fetch user from auth token failed"))
}

func LogInvalidAdmin(c *gin.Context) {
	LogError(c, fmt.Errorf("fetch admin from auth token failed"))
}

func ErrorNotFound(c *gin.Context, id int64) {
	LogError(c, fmt.Errorf("record not found: %d", id))
}
