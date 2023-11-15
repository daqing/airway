package api_resp

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type ErrCode int

const ErrGeneral ErrCode = 10000

func ErrorCodeMsg(c *gin.Context, code ErrCode, message string) {
	c.JSON(200, gin.H{"code": code, "data": gin.H{}, "message": message})
	c.Abort()
}

func ErrorCode(c *gin.Context, code ErrCode, err error) {
	ErrorCodeMsg(c, code, err.Error())
}

func Error(c *gin.Context, err error) {
	ErrorCode(c, ErrGeneral, err)
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
