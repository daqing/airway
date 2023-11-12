package hello_api

import (
	"github.com/daqing/airway/lib/api_resp"
	"github.com/gin-gonic/gin"
)

func PingAction(c *gin.Context) {
	api_resp.OK(c, gin.H{"hello": "pong"})
}
