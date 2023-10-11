package resp

import (
	"log"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data any) {
	log.Printf("OK: %+v\n", data)

	c.JSON(200, gin.H{"ok": true, "data": data, "message": ""})
}
