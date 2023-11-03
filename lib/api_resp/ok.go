package api_resp

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data any) {
	fmt.Printf("===> Response OK: %+v\n", data)

	c.JSON(200, gin.H{"ok": true, "data": data, "message": ""})
}
