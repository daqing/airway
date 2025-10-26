package render

import (
	"github.com/gin-gonic/gin"
)

const ErrGeneral int = 10000

func Error(c *gin.Context, err error) {
	ErrorCode(c, ErrGeneral, err)
}

func ErrorCode(c *gin.Context, code int, err error) {
	ErrorCodeMsg(c, code, err.Error())
}

func ErrorMessage(c *gin.Context, message string) {
	ErrorCodeMsg(c, ErrGeneral, message)
}

func ErrorCodeMsg(c *gin.Context, code int, message string) {
	c.JSON(200, gin.H{"code": code, "data": nil, "message": message})
	c.Abort()
}
