package hello_plugin

import (
	"github.com/daqing/airway/lib/resp"
	"github.com/gin-gonic/gin"
)

func PingAction(c *gin.Context) {
	resp.OK(c, gin.H{"hello": "pong"})
}
