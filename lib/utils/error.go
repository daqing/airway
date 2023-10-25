package utils

import (
	"fmt"

	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

const DefaultMessage = "===> Got server error: "

func LogError(c *gin.Context, err error) {
	LogErrorMsg(c, err, DefaultMessage)
}

func LogErrorMsg(c *gin.Context, err error, message string) {
	fmt.Println(message, err.Error())

	resp.Error(c, err)

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
