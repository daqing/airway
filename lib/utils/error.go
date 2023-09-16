package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

const DefaultMessage = "[Error for gin.Context]"

func LogError(c *gin.Context, err error) {
	LogErrorMsg(c, err, DefaultMessage)
}

func LogErrorMsg(c *gin.Context, err error, message string) {
	fmt.Println(message, "Got error: ", err)

	c.JSON(500, gin.H{"error": err.Error()})
	panic(message)
}
