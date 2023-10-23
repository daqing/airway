package utils

import "github.com/gin-gonic/gin"

const TOKEN_HEADER string = "X-Auth-Token"

func AuthToken(c *gin.Context) string {
	return c.GetHeader(TOKEN_HEADER)
}
