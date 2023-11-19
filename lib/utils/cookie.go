package utils

import "github.com/gin-gonic/gin"

func CookieToken(c *gin.Context) (string, error) {
	return c.Cookie("user_api_token")
}
