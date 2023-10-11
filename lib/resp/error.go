package resp

import (
	"log"

	"github.com/gin-gonic/gin"
)

func ErrorMsg(c *gin.Context, msg string) {
	log.Println("Error:", msg)

	c.JSON(200, gin.H{"ok": false, "data": gin.H{}, "message": msg})
	c.Abort()
}

func Error(c *gin.Context, err error) {
	ErrorMsg(c, err.Error())
}
